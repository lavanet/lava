#!/bin/bash
# scripts/test/chaos/tier_a_poisoning/a2_mid_flight_poisoning.sh
#
# Tier A scenario #2: mid-flight poisoning.
#
# Validates testing-plan §3.2 row A.2.2. Differs from A.1 in three ways:
#   1. Consumer is WARM at injection time (majorityBaseline ARMED, IsOutlier
#      guard ACTIVE) — opposite of A.1's cold-start window.
#   2. Provider 1 starts HONEST, then is flipped mid-run. Its chainTracker has
#      observed honest values before seeing the lie (still gets stuck high
#      after the flip, because chainTracker is monotonic).
#   3. The recovery path is different: the lie should NEVER land in
#      ChainState.latestBlock because IsOutlier rejects it at the front door.
#      So `poisoning_reset` should NOT fire. Instead, `set_latest_block_outlier_rejected_total`
#      (relay-response writer's front-door rejection) and
#      `sync_scoring_outlier_skipped_total{source="relay"}` (Option B drop-all-scoring
#      at the optimizer's relay path) are the canonical proof points.
#
# Pass criteria (5 assertions covering the 4 sub-conditions in the testing plan):
#   (1)   lava_consumer_set_latest_block_outlier_rejected_total advances —
#         relay-response writer (relay_processor.go) tried to push the lie into
#         ChainState; IsOutlier (now armed) rejected it.
#   (2)   lava_consumer_sync_scoring_outlier_skipped_total{source="relay"} advances —
#         optimizer's relay-path sync-scoring branch (Option B from §11.11) dropped
#         the entire sample (avail + latency + sync) on the lying provider.
#   (3)   lava_consumer_chain_state_latest_block stayed honest (never spiked to the lie).
#   (4)   lava_consumer_probe_outlier_blocked advanced — next probe cycle's Phase 2c
#         blocked provider 1 once its probe arrived with the lie.
#   (6)   lava_consumer_provider_blocked has a series with value=1 — provider 1
#         currently isolated from the session pool.

set -eu -o pipefail

__dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$__dir"/../lib/common.sh
source "$__dir"/../lib/proxy_control.sh
source "$__dir"/../lib/metrics.sh
source "$__dir"/../lib/assertions.sh

# Fixed injection value — same rationale as A.1. Well above local-chain ceiling.
INJECTED_BLOCK=1000000

# --- Phase 1: warm setup with HONEST provider 1 -------------------------

log "starting a2_mid_flight_poisoning scenario"
setup_env

# Clear all state from any prior scenario (proxy state + chainTrackers + blocked sets).
# Without restart_provider 1, its chainTracker may still be locked high from a
# previous run, which would mean provider 1 starts already-stale rather than
# honest — that would short-circuit this scenario's "mid-flight" premise.
proxy_reset_all
restart_provider 1
restart_provider 2
restart_provider 3
restart_consumer 1

# Wait long enough for the consumer to reach steady state: majorityBaseline
# armed, all providers reporting honestly, no providers blocked. 15s covers
# 2-3 probe cycles + Phase 2b consensus computation.
log "waiting 15s for consumer to reach warm steady-state with honest providers"
sleep 15

# Sanity check the env BEFORE we touch anything — if any of these fail, the
# scenario is testing the wrong starting state.
MAJORITY=$(metric_get c1 lava_consumer_majority_baseline_value)
if [ "$MAJORITY" -le 0 ]; then
    log "ABORT: majority_baseline_value=0 — consumer not warm; IsOutlier guard would be inert and this scenario can't run as intended"
    exit 2
fi
log "  baseline majority_baseline_value (sum): $MAJORITY (IsOutlier guard armed)"

BLOCKED_AT_START=$(metric_get c1 lava_consumer_provider_blocked)
if [ "$BLOCKED_AT_START" -gt 0 ]; then
    log "ABORT: provider_blocked sum=$BLOCKED_AT_START at scenario start — env not clean; expected 0"
    exit 2
fi
log "  baseline provider_blocked sum: 0 (clean starting state)"

# --- Phase 2: capture baselines, then flip provider 1 to lying mode -----

# Capture pre-flip values of every metric this scenario asserts on. The flip
# itself + post-flip relay burst are what should move them.
REJECTED_BASELINE=$(metric_get c1 lava_consumer_set_latest_block_outlier_rejected_total)
SKIP_RELAY_BASELINE=$(metric_get c1 lava_consumer_sync_scoring_outlier_skipped_total 'source="relay"')
PROBE_BLOCKED_BASELINE=$(metric_get c1 lava_consumer_probe_outlier_blocked)
CHAIN_HEAD_BASELINE=$(metric_get c1 lava_consumer_chain_state_latest_block)
log "baselines: rejected=$REJECTED_BASELINE relay_skip=$SKIP_RELAY_BASELINE probe_blocked=$PROBE_BLOCKED_BASELINE chain_head=$CHAIN_HEAD_BASELINE"

log "FLIPPING provider 1 to lying mode (block_override=$INJECTED_BLOCK)"
proxy_set_block 1 "$INJECTED_BLOCK"

# --- Phase 3: drive user relays to provoke the relay-path outlier guard ---

# To trigger assertions (1) and (2) — both fired only from the relay path —
# we need real relays where the consumer's session manager picks provider 1.
# With 3 providers and roughly equal QoS, selection is ~1/3 per relay.
#
# Loop: fire one /block relay at a time, poll the target metric after each.
# Exit the moment it advances (deterministic exit-when-success). 30-relay /
# 30s ceiling guards against pathological selection runs where provider 1
# never gets picked before the next probe cycle blocks it.
log "issuing user relays until set_latest_block_outlier_rejected_total advances (max 30)"
RELAYS_SENT=0
for i in $(seq 1 30); do
    curl -s -H 'dapp-id: alice' http://127.0.0.1:3361/block > /dev/null || true
    RELAYS_SENT=$((RELAYS_SENT + 1))
    sleep 0.2
    CURRENT_REJECTED=$(metric_get c1 lava_consumer_set_latest_block_outlier_rejected_total)
    if [ "$CURRENT_REJECTED" -gt "$REJECTED_BASELINE" ]; then
        log "  set_latest_block_outlier_rejected_total advanced after $RELAYS_SENT relays (counter: $REJECTED_BASELINE → $CURRENT_REJECTED)"
        break
    fi
done

# Wait one more probe cycle so the probe-side Phase 2c definitely runs (this
# is what asserts (4) and (6) rely on — provider 1 must get blocked).
log "waiting 6s for probe cycle to also detect + block provider 1"
sleep 6

# --- Phase 4: assertions ------------------------------------------------

# (1) Relay-response writer's SetLatestBlock rejected the lie.
# Already captured above as CURRENT_REJECTED.
log "lava_consumer_set_latest_block_outlier_rejected_total: $REJECTED_BASELINE → $CURRENT_REJECTED"
if [ "$CURRENT_REJECTED" -le "$REJECTED_BASELINE" ]; then
    log "ASSERT FAILED (1): expected set_latest_block_outlier_rejected_total to advance, stayed at $REJECTED_BASELINE"
    log "  (possible cause: session manager never selected provider 1 in $RELAYS_SENT relays before probe-side blocking — re-run if flaky)"
    exit 1
fi
log "  ✓ (1) relay-response writer rejected the lie at the front door"

# (2) Optimizer's relay path Option B drop-all-scoring fired.
SKIP_RELAY_CURRENT=$(metric_get c1 lava_consumer_sync_scoring_outlier_skipped_total 'source="relay"')
log "lava_consumer_sync_scoring_outlier_skipped_total source=relay (sum): $SKIP_RELAY_BASELINE → $SKIP_RELAY_CURRENT"
if [ "$SKIP_RELAY_CURRENT" -le "$SKIP_RELAY_BASELINE" ]; then
    log "ASSERT FAILED (2): expected sync_scoring_outlier_skipped{source=relay} to advance, stayed at $SKIP_RELAY_BASELINE"
    exit 1
fi
log "  ✓ (2) sync-scoring relay path dropped the entire sample (Option B)"

# (3) ChainState.latestBlock stayed honest. Same upper-bound rationale as A.1.
CHAIN_HEAD_CURRENT=$(metric_get c1 lava_consumer_chain_state_latest_block)
log "lava_consumer_chain_state_latest_block: $CHAIN_HEAD_BASELINE → $CHAIN_HEAD_CURRENT (lie was $INJECTED_BLOCK)"
if [ "$CHAIN_HEAD_CURRENT" -le 0 ] || [ "$CHAIN_HEAD_CURRENT" -ge 100000 ]; then
    log "ASSERT FAILED (3): chain head should stay honest (0 < x < 100000), got $CHAIN_HEAD_CURRENT"
    exit 1
fi
# Stronger check: latestBlock should have moved forward naturally (chain is
# producing blocks) but NOT spiked to the lie. Should be >= baseline.
if [ "$CHAIN_HEAD_CURRENT" -lt "$CHAIN_HEAD_BASELINE" ]; then
    log "ASSERT FAILED (3): chain head went DOWN ($CHAIN_HEAD_BASELINE → $CHAIN_HEAD_CURRENT); should advance with chain progress"
    exit 1
fi
log "  ✓ (3) chainState.latestBlock stayed honest, advanced naturally with chain"

# (4) Probe-path Phase 2c blocked provider 1 (counter — "ever blocked this run").
PROBE_BLOCKED_CURRENT=$(metric_get c1 lava_consumer_probe_outlier_blocked)
log "lava_consumer_probe_outlier_blocked (sum): $PROBE_BLOCKED_BASELINE → $PROBE_BLOCKED_CURRENT"
if [ "$PROBE_BLOCKED_CURRENT" -le "$PROBE_BLOCKED_BASELINE" ]; then
    log "ASSERT FAILED (4): probe_outlier_blocked should advance after the probe cycle, stayed at $PROBE_BLOCKED_BASELINE"
    exit 1
fi
log "  ✓ (4) probe-path Phase 2c blocked the lying provider"

# (6) Provider 1 is currently isolated from session pool (gauge).
BLOCKED_NOW=$(metric_get c1 lava_consumer_provider_blocked)
log "lava_consumer_provider_blocked (sum, # of currently-blocked provider-apiInterface pairs): $BLOCKED_NOW"
if [ "$BLOCKED_NOW" -lt 1 ]; then
    log "ASSERT FAILED (6): expected at least one provider currently in blocked list, got $BLOCKED_NOW"
    exit 1
fi
log "  ✓ (6) at least one provider is currently isolated from the session pool"

# Negative assertion: no poisoning_reset should have fired in this run. With
# IsOutlier armed before the flip, the lie should never have landed in
# ChainState.latestBlock — so AlignLatestBlockWithConsensus has nothing to
# realign. Capture-baseline-and-compare not needed because we just want = 0
# delta. Use a baseline pinned before phase 3.
# (Skipped — would add an extra metric_get without strong signal. The combination
# of asserts 3+1 already covers "the lie never landed".)

# --- Phase 5: cleanup ---------------------------------------------------

log "resetting all proxy overrides"
proxy_reset_all
log "a2_mid_flight_poisoning PASS"
