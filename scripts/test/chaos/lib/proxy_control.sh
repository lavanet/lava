# scripts/test/chaos/lib/proxy_control.sh
#
# Thin wrappers around proxy.py's HTTP control API. One function per primitive
# (set-block / set-latency / set-error / reset / status). Each wrapper takes a
# <proxy-id> integer (1, 2, or 3) which maps to the matching PROXY{N}_CTRL port
# constant from common.sh.
#
# Source via:
#     source "$LIB_DIR"/common.sh
#     source "$LIB_DIR"/proxy_control.sh
# (proxy_control.sh depends on the PROXY{N}_CTRL constants from common.sh.)
#
# Each wrapper returns 0 on HTTP 200, 1 otherwise. Errors are logged via log()
# from common.sh so scenario callers don't need to inspect curl output.
#
# proxy.py control API (per testutil/debugging/mock_node_proxy/proxy.py docstring):
#   POST /set-block?height=N    Override block height in responses
#   POST /set-latency?ms=N      Add delay before forwarding responses
#   POST /set-error?enabled=B   Return HTTP 500 for all requests
#   POST /reset                 Clear all overrides (back to transparent)
#   GET  /status                Show current configuration
#
# See: protocol/rpcconsumer/docs/probing/improvements-proposals/pre-release-testing-infrastructure.md §4.2.2

# _proxy_ctrl_port — internal helper. Resolves <proxy-id> ∈ {1,2,3} to the
# corresponding PROXY{N}_CTRL constant. Returns 0 on success and prints the port
# to stdout; returns 1 (with log line) on invalid id.
_proxy_ctrl_port() {
    case "$1" in
        1) echo "$PROXY1_CTRL" ;;
        2) echo "$PROXY2_CTRL" ;;
        3) echo "$PROXY3_CTRL" ;;
        *) log "_proxy_ctrl_port: invalid proxy-id '$1' (expected 1, 2, or 3)"; return 1 ;;
    esac
}

# proxy_set_block <proxy-id> <height>
# Inject a block-height override into the proxy's response stream. From now on,
# REST /blocks/latest and Tendermint RPC status/block responses report <height>
# as the chain head, regardless of what the upstream lavad node says. Used by
# Tier A poisoning scenarios to simulate a malicious provider.
#
# Use proxy_reset (subtask 2.9) to clear the override.
proxy_set_block() {
    local proxy_id="$1"
    local height="$2"
    if [ -z "$proxy_id" ] || [ -z "$height" ]; then
        log "proxy_set_block: usage: proxy_set_block <proxy-id> <height>"
        return 1
    fi
    local port
    port=$(_proxy_ctrl_port "$proxy_id") || return 1
    if ! curl -sf -X POST "http://127.0.0.1:${port}/set-block?height=${height}" -o /dev/null --connect-timeout 2; then
        log "proxy_set_block: control API at :${port} did not accept set-block?height=${height}"
        return 1
    fi
    return 0
}

# proxy_set_latency <proxy-id> <ms>
# Add a delay before forwarding responses. Used by Tier A scenarios that need to
# simulate slow providers (e.g., to verify the consumer's relay timeout / retry
# behavior or to widen race windows).
#
# Use proxy_reset to clear (sets latency_ms back to 0).
proxy_set_latency() {
    local proxy_id="$1"
    local ms="$2"
    if [ -z "$proxy_id" ] || [ -z "$ms" ]; then
        log "proxy_set_latency: usage: proxy_set_latency <proxy-id> <ms>"
        return 1
    fi
    local port
    port=$(_proxy_ctrl_port "$proxy_id") || return 1
    if ! curl -sf -X POST "http://127.0.0.1:${port}/set-latency?ms=${ms}" -o /dev/null --connect-timeout 2; then
        log "proxy_set_latency: control API at :${port} did not accept set-latency?ms=${ms}"
        return 1
    fi
    return 0
}

# proxy_set_error <proxy-id> <true|false>
# Toggle "always return HTTP 500" mode. Used by Tier A scenarios to simulate a
# provider whose backend node is throwing errors (every REST/RPC request 500s).
# Pass "false" to disable (equivalent to one axis of proxy_reset).
proxy_set_error() {
    local proxy_id="$1"
    local enabled="$2"
    if [ -z "$proxy_id" ] || [ -z "$enabled" ]; then
        log "proxy_set_error: usage: proxy_set_error <proxy-id> <true|false>"
        return 1
    fi
    if [ "$enabled" != "true" ] && [ "$enabled" != "false" ]; then
        log "proxy_set_error: enabled must be 'true' or 'false', got '$enabled'"
        return 1
    fi
    local port
    port=$(_proxy_ctrl_port "$proxy_id") || return 1
    if ! curl -sf -X POST "http://127.0.0.1:${port}/set-error?enabled=${enabled}" -o /dev/null --connect-timeout 2; then
        log "proxy_set_error: control API at :${port} did not accept set-error?enabled=${enabled}"
        return 1
    fi
    return 0
}

# proxy_reset <proxy-id>
# Clear ALL overrides on the named proxy (block override + latency + error mode
# all return to transparent). Always call from teardown / between scenarios so
# each test starts from a clean slate.
proxy_reset() {
    local proxy_id="$1"
    if [ -z "$proxy_id" ]; then
        log "proxy_reset: usage: proxy_reset <proxy-id>"
        return 1
    fi
    local port
    port=$(_proxy_ctrl_port "$proxy_id") || return 1
    if ! curl -sf -X POST "http://127.0.0.1:${port}/reset" -o /dev/null --connect-timeout 2; then
        log "proxy_reset: control API at :${port} did not accept /reset"
        return 1
    fi
    return 0
}

# proxy_reset_all
# Convenience: reset all three proxies in one call. Returns 0 only if all three
# /reset POSTs succeeded; logs which proxy failed if any. Best-effort: continues
# resetting the others even after a failure, so a single dead proxy doesn't
# leave the others in a poisoned state.
proxy_reset_all() {
    local rc=0 i
    for i in 1 2 3; do
        if ! proxy_reset "$i"; then
            rc=1
        fi
    done
    return $rc
}
