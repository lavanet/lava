# scripts/test/chaos/lib/common.sh
#
# Shared constants + utility functions for the pre-release chaos test suite.
# Source from any scenario script:
#
#   __dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
#   source "$__dir"/../lib/common.sh
#
# This file mirrors the port + path layout in
#   scripts/pre_setups/init_lava_chaos_test_env.sh
# Keep them in sync. If you change a port there, change it here too.
#
# See: protocol/rpcconsumer/docs/probing/improvements-proposals/pre-release-testing-infrastructure.md §4.2

# --- Repo paths ------------------------------------------------------------

# Resolve repo root from this file's location: scripts/test/chaos/lib/ -> repo root is 4 levels up.
LIB_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
REPO_ROOT="$( cd -- "$LIB_DIR/../../../.." &> /dev/null && pwd )"
INIT_SCRIPT="$REPO_ROOT/scripts/pre_setups/init_lava_chaos_test_env.sh"
LOGS_DIR="$REPO_ROOT/testutil/debugging/logs"

# --- Cache server (one shared instance) -----------------------------------

CACHE_LISTEN_PORT=20100
CACHE_METRICS_PORT=20200

# --- proxy.py ports (one triple per provider) -----------------------------
# REST/RPC are the data ports providers point at; CTRL is the proxy's HTTP control API.

PROXY1_REST=4000 ; PROXY1_RPC=4010 ; PROXY1_CTRL=4001
PROXY2_REST=4100 ; PROXY2_RPC=4110 ; PROXY2_CTRL=4101
PROXY3_REST=4200 ; PROXY3_RPC=4210 ; PROXY3_CTRL=4201

# --- Provider ports (one listener + one metrics port per provider) --------

PROVIDER1_LISTENER="127.0.0.1:2220" ; PROVIDER1_METRICS_PORT=7766
PROVIDER2_LISTENER="127.0.0.1:2221" ; PROVIDER2_METRICS_PORT=7756
PROVIDER3_LISTENER="127.0.0.1:2222" ; PROVIDER3_METRICS_PORT=7746

# --- Consumer ports (rest + tmrpc + grpc + metrics, two consumers) --------

CONSUMER1_REST_PORT=3360 ; CONSUMER1_TMRPC_PORT=3361 ; CONSUMER1_GRPC_PORT=3362 ; CONSUMER1_METRICS_PORT=7779
CONSUMER2_REST_PORT=3460 ; CONSUMER2_TMRPC_PORT=3461 ; CONSUMER2_GRPC_PORT=3462 ; CONSUMER2_METRICS_PORT=7780

# Metrics-port aliases: scenario scripts read METRICS_PORT_C{1,2} consistent with the plan's spec.
METRICS_PORT_C1=$CONSUMER1_METRICS_PORT
METRICS_PORT_C2=$CONSUMER2_METRICS_PORT

# --- Functions ------------------------------------------------------------

# Names of the 10 screens the chaos env launches.
EXPECTED_SCREENS=(node cache_consumer proxy1 proxy2 proxy3 provider1 provider2 provider3 consumers_1 consumers_2)

# setup_env — bring the chaos env up if it's not already running, then wait until
# consumer 1 is serving relays (the surest "everything is ready" signal — consumer's
# metrics endpoint binds early but probe-derived metrics only appear after providers
# answer probes, so we wait on /metrics for substantive content).
#
# Idempotent: if all 10 expected screens are already present, returns immediately
# so scenario scripts can be re-run against a live env without a tear-down/rebuild
# cycle. Tear-down via teardown(); skip and re-source if scenarios need a clean env.
setup_env() {
    local ls_output running_count
    ls_output=$(screen -ls 2>/dev/null || true)
    running_count=0
    for s in "${EXPECTED_SCREENS[@]}"; do
        echo "$ls_output" | grep -qE "[0-9]+\.${s}\b" && running_count=$((running_count + 1))
    done

    if [ "$running_count" -eq "${#EXPECTED_SCREENS[@]}" ]; then
        echo "[common.sh] env already up (${running_count}/${#EXPECTED_SCREENS[@]} screens) — reusing" >&2
        return 0
    fi

    echo "[common.sh] env not fully up (${running_count}/${#EXPECTED_SCREENS[@]} screens) — launching $INIT_SCRIPT" >&2
    if [ ! -x "$INIT_SCRIPT" ]; then
        echo "[common.sh] ERROR: init script not executable: $INIT_SCRIPT" >&2
        return 1
    fi
    # init script chdir's via __dir, but it expects pwd at repo root (relative paths
    # like ./scripts/start_env_dev.sh + specs/...). Run from REPO_ROOT to be safe.
    ( cd "$REPO_ROOT" && "$INIT_SCRIPT" ) >/dev/null 2>&1
    local init_rc=$?
    if [ $init_rc -ne 0 ]; then
        echo "[common.sh] ERROR: init script exited $init_rc" >&2
        return 1
    fi

    # Wait for proxy control APIs (fast — they bind in <1s after launch).
    local i ctrl_port
    for i in 1 2 3; do
        case $i in
            1) ctrl_port=$PROXY1_CTRL ;;
            2) ctrl_port=$PROXY2_CTRL ;;
            3) ctrl_port=$PROXY3_CTRL ;;
        esac
        if ! wait_until "curl -sf -o /dev/null --connect-timeout 1 http://127.0.0.1:${ctrl_port}/status" 30; then
            echo "[common.sh] ERROR: proxy ${i} control API (:${ctrl_port}) not responding after 30s" >&2
            return 1
        fi
    done
    echo "[common.sh] all 3 proxy control APIs up" >&2

    # Wait for consumer 1's relay path. /metrics binds early (just an http handler),
    # so we additionally require `lava_consumer_provider_liveness` = 1 — this metric
    # only emits after a successful probe round, so it's a strict "everything is ready"
    # signal. 2s poll interval keeps log noise low (probes run every 5s anyway).
    local liveness_check="curl -sf --connect-timeout 1 http://127.0.0.1:${METRICS_PORT_C1}/metrics 2>/dev/null | grep -qE '^lava_consumer_provider_liveness\\{[^}]+\\} 1\\b'"
    if ! wait_until "$liveness_check" 90 2; then
        echo "[common.sh] ERROR: consumer 1 did not advance provider_liveness within 90s — env may be partially broken" >&2
        return 1
    fi
    echo "[common.sh] consumer 1 ready (provider_liveness=1 observed at :${METRICS_PORT_C1}/metrics)" >&2
    return 0
}

# restart_consumer <id>
# Kill consumer <id>'s screen (1 or 2), truncate its log, and relaunch it with
# the same command line the init script uses. Waits until provider_liveness=1
# comes back (i.e. the consumer has rejoined the probe consensus). ~10s typical.
#
# Why this exists: Phase 2c (`blockOutlierProviders`) blocks a provider exactly
# once per epoch when first detected as an outlier — re-blocking is a no-op for
# the metric. This makes scenarios that rely on "counter advanced" assertions
# non-idempotent across runs against the same env. Restarting just the consumer
# clears its in-memory blocked-providers map without the ~90s cost of a full
# env teardown + setup.
#
# IMPORTANT: the relaunch command must stay in sync with the corresponding
# `screen -d -m -S consumers_${id}` invocation in
# `scripts/pre_setups/init_lava_chaos_test_env.sh`. If you add a flag there,
# add it here too. (Future cleanup: factor the consumer launch into a shared
# helper that both files source.)
restart_consumer() {
    local id="$1"
    local rest_port tmrpc_port grpc_port metrics_port screen_name log_file
    case "$id" in
        1)
            rest_port=$CONSUMER1_REST_PORT
            tmrpc_port=$CONSUMER1_TMRPC_PORT
            grpc_port=$CONSUMER1_GRPC_PORT
            metrics_port=$CONSUMER1_METRICS_PORT
            screen_name="consumers_1"
            log_file="$LOGS_DIR/CONSUMERS_1.log"
            ;;
        2)
            rest_port=$CONSUMER2_REST_PORT
            tmrpc_port=$CONSUMER2_TMRPC_PORT
            grpc_port=$CONSUMER2_GRPC_PORT
            metrics_port=$CONSUMER2_METRICS_PORT
            screen_name="consumers_2"
            log_file="$LOGS_DIR/CONSUMERS_2.log"
            ;;
        *)
            log "restart_consumer: invalid consumer-id '$id' (expected 1 or 2)"
            return 1
            ;;
    esac

    echo "[common.sh] restarting consumer $id ($screen_name)" >&2
    screen -X -S "$screen_name" quit 2>/dev/null || true
    # `screen -X quit` kills the screen wrapper but the inner `lavap | tee` pipe
    # orphans lavap to PID 1, where it keeps holding the metrics port. The NEW
    # lavap would then die on EADDRINUSE and our liveness check would falsely
    # pass against the OLD process. Explicitly kill anything still bound to the
    # consumer's metrics port, then wait for the port to free up.
    local stale_pids
    stale_pids=$(lsof -ti ":${metrics_port}" 2>/dev/null || true)
    if [ -n "$stale_pids" ]; then
        echo "[common.sh] killing stale lavap PIDs holding :${metrics_port}: $stale_pids" >&2
        # shellcheck disable=SC2086
        kill -9 $stale_pids 2>/dev/null || true
    fi
    if ! wait_until "! lsof -ti ':${metrics_port}' >/dev/null 2>&1" 10; then
        echo "[common.sh] ERROR: port :${metrics_port} still occupied 10s after kill" >&2
        return 1
    fi
    : > "$log_file"

    # Mirrors init_lava_chaos_test_env.sh's consumer launch (minus $EXTRA_PORTAL_FLAGS,
    # which is empty by default per scripts/vars/variables.sh; if operators set it,
    # they should restart the full env instead).
    ( cd "$REPO_ROOT" && screen -d -m -S "$screen_name" bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:${rest_port} LAV1 rest 127.0.0.1:${tmrpc_port} LAV1 tendermintrpc 127.0.0.1:${grpc_port} LAV1 grpc \
--geolocation 1 --log_level trace --from user1 --chain-id lava --shared-state --cache-be 127.0.0.1:${CACHE_LISTEN_PORT} --allow-insecure-provider-dialing --metrics-listen-address \":${metrics_port}\" --debug-probes --enable-periodic-probe-providers --majority-baseline-bucket-time-window 2m 2>&1 | tee ${log_file}" )
    sleep 0.25

    local liveness_check="curl -sf --connect-timeout 1 http://127.0.0.1:${metrics_port}/metrics 2>/dev/null | grep -qE '^lava_consumer_provider_liveness\\{[^}]+\\} 1\\b'"
    if ! wait_until "$liveness_check" 60 2; then
        echo "[common.sh] ERROR: consumer $id did not advance provider_liveness within 60s after restart" >&2
        return 1
    fi
    echo "[common.sh] consumer $id ready (provider_liveness=1)" >&2
    return 0
}

# restart_provider <id>
# Kill provider <id>'s screen (1, 2, or 3), truncate its log, and relaunch it
# with the same command line the init script uses (still pointing at its proxy).
# Waits until "RPCProvider pubkey:" appears in the new log, signalling startup
# is complete (typical ~5s).
#
# Why this exists: provider lavap's `ChainTracker` is monotonic — once it has
# observed a block height, it never decreases (even if the upstream node now
# reports lower values). After a `proxy_set_block <id> 108000` injection,
# resetting the proxy doesn't unstick the provider's chainTracker; the
# provider becomes permanently "stale" relative to the real chain. Restarting
# the provider clears the chainTracker in-memory and lets it re-read fresh
# values from the (now-transparent) proxy.
#
# IMPORTANT: like restart_consumer, the relaunch command must stay in sync
# with init_lava_chaos_test_env.sh. See the corresponding `screen -d -m -S
# provider${id}` invocation there.
restart_provider() {
    local id="$1"
    local listener metrics_port proxy_rest proxy_rpc screen_name log_file servicer
    case "$id" in
        1) listener=$PROVIDER1_LISTENER; metrics_port=$PROVIDER1_METRICS_PORT; proxy_rest=$PROXY1_REST; proxy_rpc=$PROXY1_RPC; screen_name="provider1"; log_file="$LOGS_DIR/PROVIDER1.log"; servicer="servicer1" ;;
        2) listener=$PROVIDER2_LISTENER; metrics_port=$PROVIDER2_METRICS_PORT; proxy_rest=$PROXY2_REST; proxy_rpc=$PROXY2_RPC; screen_name="provider2"; log_file="$LOGS_DIR/PROVIDER2.log"; servicer="servicer2" ;;
        3) listener=$PROVIDER3_LISTENER; metrics_port=$PROVIDER3_METRICS_PORT; proxy_rest=$PROXY3_REST; proxy_rpc=$PROXY3_RPC; screen_name="provider3"; log_file="$LOGS_DIR/PROVIDER3.log"; servicer="servicer3" ;;
        *) log "restart_provider: invalid provider-id '$id' (expected 1, 2, or 3)"; return 1 ;;
    esac

    # The provider's tendermintrpc launch arg needs $LAVA_RPC_WS (and grpc needs
    # $LAVA_GRPC) from scripts/vars/variables.sh. Source it so the function works
    # from any caller's environment.
    # shellcheck disable=SC1091
    source "$REPO_ROOT/scripts/vars/variables.sh"

    echo "[common.sh] restarting provider $id ($screen_name)" >&2
    screen -X -S "$screen_name" quit 2>/dev/null || true
    # Same pattern as restart_consumer: kill the orphaned lavap process holding
    # the listener port (the `tee` pipe outlives screen).
    local listen_port=${listener##*:}
    local stale_pids
    stale_pids=$(lsof -ti ":${listen_port}" 2>/dev/null || true)
    if [ -n "$stale_pids" ]; then
        echo "[common.sh] killing stale lavap PIDs holding :${listen_port}: $stale_pids" >&2
        # shellcheck disable=SC2086
        kill -9 $stale_pids 2>/dev/null || true
    fi
    if ! wait_until "! lsof -ti ':${listen_port}' >/dev/null 2>&1" 10; then
        echo "[common.sh] ERROR: port :${listen_port} still occupied 10s after kill" >&2
        return 1
    fi
    : > "$log_file"

    ( cd "$REPO_ROOT" && screen -d -m -S "$screen_name" bash -c "source ~/.bashrc; lavap rpcprovider \
${listener} LAV1 rest 'http://127.0.0.1:${proxy_rest}' \
${listener} LAV1 tendermintrpc 'http://127.0.0.1:${proxy_rpc},${LAVA_RPC_WS}' \
${listener} LAV1 grpc '${LAVA_GRPC}' \
--geolocation 1 --log_level debug --from ${servicer} --chain-id lava --metrics-listen-address \":${metrics_port}\" 2>&1 | tee ${log_file}" )
    sleep 0.25

    # Wait for the "RPCProvider pubkey:" log line — strongest "startup complete"
    # signal in the provider's boot sequence.
    if ! wait_until "grep -q 'RPCProvider pubkey:' '${log_file}'" 30 2; then
        echo "[common.sh] ERROR: provider $id did not finish startup within 30s after restart" >&2
        return 1
    fi
    echo "[common.sh] provider $id ready" >&2
    return 0
}

# teardown — kill all screens + wipe stale socket entries.
# Idempotent: safe to call multiple times, safe to call when no screens are running.
# Mirrors the pattern at the top of init_lava_chaos_test_env.sh ("killall screen; screen -wipe").
#
# Intended usage at the top of every scenario script:
#     trap teardown EXIT
# so that an unexpected exit (assertion failure, ctrl-C, error in setup) still leaves
# the host clean. Scenarios that want to KEEP the env across runs (faster iteration)
# should NOT install the trap — setup_env() is already idempotent on a live env.
teardown() {
    # killall returns non-zero if there are no matching processes; suppress that.
    # Send SIGTERM first (graceful screen detach), wait briefly, then SIGKILL anything
    # that survived. screen -wipe cleans up sockets whose owning process is gone.
    killall screen 2>/dev/null || true
    # Brief grace period — screen processes flush their session state on SIGTERM.
    sleep 1
    killall -9 screen 2>/dev/null || true
    screen -wipe >/dev/null 2>&1 || true
    echo "[common.sh] teardown complete" >&2
}

# log <msg> — timestamped, scenario-tagged stderr log line.
# Output: [2026-05-10T17:42:13] [smoke] msg
# Writes to stderr so scenario stdout stays clean for piped consumers (e.g. JSON
# results, metric values produced by the scenario).
#
# Scenario name is auto-derived from the calling script's filename (without `.sh`
# extension); override by setting $SCENARIO_NAME before sourcing this lib if $0
# isn't appropriate (e.g. when running via `bash -c`).
log() {
    local scenario_name ts
    scenario_name="${SCENARIO_NAME:-$(basename "$0" .sh)}"
    ts=$(date +"%Y-%m-%dT%H:%M:%S")
    echo "[$ts] [$scenario_name] $*" >&2
}

# wait_until <cmd> <timeout-sec> [<interval-sec>]
#
# Polls <cmd> (any shell expression) once per <interval-sec> seconds until it
# returns 0 or <timeout-sec> elapses. Returns 0 on success, 1 on timeout.
# Default interval: 1 second.
#
# Examples:
#   wait_until 'curl -sf http://127.0.0.1:4001/status >/dev/null' 30
#   wait_until 'metric_get c1 lava_consumer_chain_state_latest_block | grep -q ^1' 60 2
#
# Note: <cmd> runs via `eval`, so it can contain pipes, &&, command substitutions,
# etc. Don't quote it as if for argv — pass the literal string. Quoting:
#   wait_until "curl -sf http://...:${PROXY1_CTRL}/status >/dev/null" 30
# (double-quoted at the call site so the variable expands once into the eval'd cmd).
wait_until() {
    local cmd="$1"
    local timeout="$2"
    local interval="${3:-1}"
    local deadline
    deadline=$(( $(date +%s) + timeout ))
    while [ "$(date +%s)" -lt "$deadline" ]; do
        if eval "$cmd"; then
            return 0
        fi
        sleep "$interval"
    done
    log "wait_until timed out after ${timeout}s: $cmd"
    return 1
}
