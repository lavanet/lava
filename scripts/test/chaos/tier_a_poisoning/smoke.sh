#!/bin/bash
# scripts/test/chaos/tier_a_poisoning/smoke.sh
#
# Phase 1 wiring-proof scenario. Not one of the 5 Tier A poisoning scenarios —
# this is the simplest possible end-to-end exercise that confirms the
# infrastructure works: setup_env brings the chaos env up, proxy_set_block
# successfully causes a malicious-block to flow into the consumer's metrics,
# and proxy_reset_all returns to transparent.
#
# Pass: this script exits 0. Fail: any non-zero exit (assertion failure, env
# bring-up failure, proxy unreachable).
#
# See: protocol/rpcconsumer/docs/probing/improvements-proposals/pre-release-testing-infrastructure.md §4.3

set -eu -o pipefail

__dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$__dir"/../lib/common.sh
source "$__dir"/../lib/proxy_control.sh
source "$__dir"/../lib/metrics.sh
source "$__dir"/../lib/assertions.sh

# Note: this scenario does NOT install `trap teardown EXIT`. Leaving the env up
# between runs is the right default for iterative development of the smoke
# scenario; once Tier A scenarios graduate to run_all.sh, the orchestrator
# decides whether to keep or tear down based on overall pass/fail.

# --- Phase 1: setup (subtask 3.1) -----------------------------------------

log "starting smoke scenario"
setup_env

# Pre-scenario state cleanup. Three sources of cross-run state pollution that
# we have to clear to make this scenario idempotent on a shared env:
#
#   1. Proxy overrides — proxy_reset_all clears any block / latency / error
#      overrides left by a prior run that crashed before reaching its own cleanup.
#   2. Provider 1's chainTracker — monotonic across the process lifetime, so a
#      prior injection (e.g. block=108243) leaves the chainTracker stuck high
#      forever, causing the provider to be permanently flagged as "stale".
#      restart_provider clears it.
#   3. Consumer 1's blocked-providers map — Phase 2c blocks a provider exactly
#      once per epoch, so a re-run wouldn't re-block the same provider and the
#      outlier counter wouldn't advance. restart_consumer clears it.
proxy_reset_all
restart_provider 1
restart_consumer 1

# Wait for two probe cycles so consumer 1's metrics reflect at least one full
# round of consensus + scoring. restart_consumer already waits until
# provider_liveness=1 (one successful probe round); this extra wait covers the
# second cycle that subsequent subtasks rely on for baseline snapshots.
log "waiting ~10s for two probe cycles to elapse"
sleep 10
log "setup complete"

# --- Phase 2: baseline snapshot (subtask 3.2) -----------------------------

# Capture two baselines:
#   BASELINE                  — current chain head, used by 3.3 to compute the
#                               injection value (BASELINE + 100000) so the lie is
#                               unambiguously above majorityBaseline + outlierThreshold.
#   OUTLIER_BLOCKED_BASELINE  — current total of probe-outlier-blocked events
#                               across all 3 providers / 3 apiInterfaces. The
#                               post-injection assertion (subtask 3.5) requires
#                               this counter to strictly exceed the baseline.
BASELINE=$(metric_get c1 lava_consumer_chain_state_latest_block)
OUTLIER_BLOCKED_BASELINE=$(metric_get c1 lava_consumer_probe_outlier_blocked)
log "baseline chain head: $BASELINE"
log "baseline probe_outlier_blocked total: $OUTLIER_BLOCKED_BASELINE"

if [ "$BASELINE" -le 0 ]; then
    log "ERROR: baseline chain head is 0 — env not warm yet, cannot proceed"
    exit 1
fi

# --- Phase 3: inject lie via proxy 1 (subtask 3.3) ------------------------

# Pick an injected value comfortably above majorityBaseline + outlierThreshold.
# Default outlierThreshold is 100 (see ComputeOutlierThreshold in chainstate);
# BASELINE + 100000 is well into the "obvious poisoning" range, mirroring the
# Feb 17 incident class (cross-chain contamination producing 20k-block jumps).
INJECTED_BLOCK=$((BASELINE + 100000))
log "injecting block override on proxy 1: $INJECTED_BLOCK (BASELINE=$BASELINE + 100000)"
proxy_set_block 1 "$INJECTED_BLOCK"
log "proxy 1 status: $(curl -s "http://127.0.0.1:${PROXY1_CTRL}/status" | tr -d '\n ')"

# --- Phase 4: wait for probe cycle (subtask 3.4) --------------------------

# Default --periodic-probe-providers-interval is 5s. Wait 6s as a safe
# over-estimate of "one full probe cycle has elapsed", per §8.4 of the
# infrastructure doc (time-based asserts use hard-coded constants since
# there's no specific predicate to poll for).
log "waiting 6s for one probe cycle to carry the lie to consumer 1"
sleep 6

# --- Phase 5: assert outlier_blocked counter advanced (subtask 3.5) -------

# Provider address is non-deterministic across init script runs (staking keys
# generate different addresses depending on key state), so we DON'T pin the
# assertion to a specific provider_address label. Instead `metric_get` with no
# labels sums across all matching series — the total advances whenever ANY
# provider is newly blocked. The provider behind proxy 1 is the only new
# blocker introduced by Phase 3, so the sum reliably advances.
log "asserting lava_consumer_probe_outlier_blocked advanced past baseline $OUTLIER_BLOCKED_BASELINE"
CURRENT_OUTLIER_BLOCKED=$(metric_get c1 lava_consumer_probe_outlier_blocked)
log "  current value: $CURRENT_OUTLIER_BLOCKED"
assert_metric_increased_since c1 lava_consumer_probe_outlier_blocked '' "$OUTLIER_BLOCKED_BASELINE"
log "  ✓ assertion passed"

# --- Phase 6: reset proxies (subtask 3.6) ---------------------------------

# Reset proxy 1 (and the others, defensively, in case a future revision uses
# them) back to transparent mode. This is the recoverable-state cleanup: the
# proxies' overrides are cleared in-place.
#
# Consumer-blocked / provider-chainTracker state is NOT cleaned up here — that
# requires process restarts, which the next run's Phase 1 already does
# (restart_provider 1 + restart_consumer 1). Doing it here too would just add
# latency. Per the plan note: "or just leave it blocked — scenario teardown
# handles it" — scenarios don't install trap teardown EXIT because keeping
# the env up is the iterative-development default.
log "resetting all proxy overrides"
proxy_reset_all

log "smoke scenario PASS"
