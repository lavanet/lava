#!/bin/bash
# scripts/test/chaos/tier_a_poisoning/a4_consensus_failure_window.sh
#
# Tier A scenario #4: consensus-failure window.
#
# Validates testing-plan §3.2 row A.2.4. Differs from A.1/A.2/A.3 in attack
# strategy: rather than ONE provider lying (which the majority detects as an
# outlier), here ALL THREE providers report wildly-different blocks. With no
# bucket containing ≥51% support, `ComputeMajorityBaseline` returns 0 → CSM
# resets `majorityBaseline=0` → the IsOutlier guard becomes INERT.
#
# This is a documented "security gap" window in the design: when consensus
# can't be computed, the unification has no floor to compare against, so the
# IsOutlier defense disables itself. The pre-release-testing-plan calls this
# out explicitly — it's accepted behavior, and the observability story
# (consensus_failure counter + majority_baseline_value=0 gauge) is what lets
# operators alert on it.
#
# Production analogue: a network partition splits providers across forks; a
# chain reorg leaves providers at meaningfully different heights; or a buggy
# release deploys to providers and confuses block reporting. In all three
# cases the consumer should refuse to "guess" a baseline rather than
# fabricate one from spread-out values.
#
# --- Defense state during the window -----------------------------------------
#
# Active:
#   ✓ Monotonic check inside SetLatestBlock (chain_state.go:85). This is the
#     WRONG defense to rely on here — it locks IN the highest landed value
#     rather than rejecting it. Once chain_state.latestBlock is 130k, any
#     subsequent honest probe at, say, head=92 is rejected by monotonicity
#     until AlignLatestBlockWithConsensus snaps it down (the recovery path
#     tested by A.5).
#   ✓ consensus_failure counter (consumer_session_manager.go:556) — the ONLY
#     operator-visible signal that this window is open. Alerting on this is
#     mandatory; alerting on outlier_rejected misses the entire scenario.
#
# Disabled (because all gate on `majorityBaseline > 0`):
#   ✗ IsOutlier guard (chain_state.go:71, 240) — returns false; allows writes
#   ✗ AlignLatestBlockWithConsensus — skipped at consumer_session_manager.go:539
#     (the call sits inside `if floor > 0`); no realignment-down can occur
#   ✗ blockOutlierProviders (Phase 2c) — same gate; no provider gets blocked
#   ✗ Optimizer's sync-scoring IsOutlier (provider_optimizer.go:230/295) —
#     samples NOT dropped; the lying providers' QoS is updated normally
#     (M6 sync_scoring_outlier_skipped does NOT advance)
#
# --- Why ALL three SetLatestBlock paths land lies during this window --------
#
# Probe path (provider_optimizer.go:235, 299 — inside AppendRelayData /
# AppendProbeRelayData): both call SetLatestBlock(syncBlock) after the
# IsOutlier check. With majorityBaseline=0 the IsOutlier check is bypassed,
# so EVERY probe value lands and the monotonic check picks the MAX.
#
# Relay path (relay_processor.go:465): per-relay response carries the
# provider's LatestBlock; relay_processor calls SetLatestBlock(blockSeen).
# Same gate, same outcome.
#
# Shared-state path (rpcconsumer_server.go:929): cache GET returns a higher
# seenBlock → SetLatestBlockFromSharedState wraps SetLatestBlock. Same gate,
# same outcome.
#
# Consequence: after one probe round with providers at (10000, 60000, 130000),
# `chain_state.latestBlock = 130000` deterministically. The 10-relay drive
# in Phase 2 below is BELT-AND-SUSPENDERS (additionally exercises the
# relay path's SetLatestBlock); it does NOT change the expected value.
#
# --- Why ComputeBucketWidth doesn't bail us out -----------------------------
#
# With `--majority-baseline-bucket-time-window 2m` and lava's ~1s block time,
# bucketWidth=120, windowWidth=240. To force 3 providers into 3 SEPARATE
# buckets, pairwise separations must be > 240. We use (10000, 60000, 130000)
# — separations 50000 and 70000, both >> 240. ComputeMajorityBaseline then
# finds bestCount=1 (each bucket holds only itself); needs majority=(3/2)+1=2;
# returns 0 → consensus failed (chain_state.go:254 algorithm).
#
# --- Pass criteria (4 assertions, A.2.4 sub-conditions) ---------------------
#
#   (1)   `lava_consumer_majority_baseline_consensus_failure` advances — proves
#         Phase 2b detected no-majority and emitted the failure metric.
#   (2)   `lava_consumer_majority_baseline_value` == 0 — proves CSM correctly
#         RESET the baseline ("stale baseline is worse than no baseline"
#         line at consumer_session_manager.go:553-555).
#   (3)   `lava_consumer_chain_state_latest_block` == LIE_3 (130000) — proves
#         the MAX provider lie landed via the probe path's SetLatestBlock
#         (and the monotonic check then locked it in against the smaller
#         lies). Tight equality assertion — anything else indicates a code
#         change that affected one of the SetLatestBlock write sites.
#   (4)   `lava_consumer_set_latest_block_outlier_rejected_total` does NOT
#         advance during the window (REJECTED_AFTER == REJECTED_BEFORE) —
#         direct proof that the IsOutlier guard is inert under no-consensus.
#         restart_consumer zeros the counter, so in practice both values are
#         0 in a clean run; the assertion uses inequality on the captured
#         baseline so it remains correct if the env carries state. This is
#         the CENTRAL piece of operator documentation: there are NO rejection
#         events during this window, so detection must rely on (1)
#         (consensus_failure counter) — alerting on outlier_rejected would
#         miss it entirely.
#
# --- Handoff to A.5 ---------------------------------------------------------
#
# After this script exits, the proxies are reset but consumer 1's
# chain_state.latestBlock is intentionally left at 130000 (the poisoned
# value). The next consensus cycle on healthy providers (which A.5 sets up)
# will fire AlignLatestBlockWithConsensus with `outcome="poisoning_reset"`
# and snap the value down to the honest floor. Reset between runs is via
# `proxy_reset_all` only — the consumer's chain_state is NOT reset
# (only `restart_consumer 1` would do that, which A.5 deliberately avoids
# so it can observe the realignment in action).

set -eu -o pipefail

__dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$__dir"/../lib/common.sh
source "$__dir"/../lib/proxy_control.sh
source "$__dir"/../lib/metrics.sh
source "$__dir"/../lib/assertions.sh

# Three wildly-different values, all WELL above any plausible local-chain head
# (the dev chain advances ~1 block/sec) and pairwise-separated by > 240
# (bucketWidth × 2 = windowWidth, the maximum span where 2 values land in the
# same bucket). Sorted high values chosen so the MAX (130k) is numerically
# distinct from any real chain head in dashboards.
LIE_1=10000
LIE_2=60000
LIE_3=130000

# --- Phase 1: setup with 3 lying proxies + cold consumer --------------------

log "starting a4_consensus_failure_window scenario"
setup_env
proxy_reset_all

# Inject lies on ALL THREE proxies BEFORE restarting providers. Same
# sequencing rationale as A.1: provider chainTrackers are monotonic, so if
# the lie isn't already in place when the chainTracker boots, the provider
# will see the honest chain head first and never recover to report the lie.
log "injecting wildly-different blocks: proxy1=$LIE_1, proxy2=$LIE_2, proxy3=$LIE_3 (pairwise separations >> bucketWidth)"
proxy_set_block 1 "$LIE_1"
proxy_set_block 2 "$LIE_2"
proxy_set_block 3 "$LIE_3"

# Cold-start all 3 providers + consumer 1. All three providers come up
# against their lying proxy → all three report a different lie from probe 1.
log "restarting all 3 providers (cold start, each picks up its proxy's lie)"
restart_provider 1
restart_provider 2
restart_provider 3
restart_consumer 1

# Wait for ~3 probe cycles. Phase 2b runs once per cycle, computes no
# majority each time, and increments the consensus_failure counter each
# time. 15s = 3 cycles at the default 5s interval.
log "waiting 15s for ~3 probe cycles to exercise the consensus-failure path"
sleep 15

# --- Phase 2: capture baselines + drive ONE relay --------------------------

# At this point, consensus_failure should be at ~3-9 (3 cycles × possibly 3
# apiInterfaces). majority_baseline_value should be 0. chain_state_latest_block
# is still ~0 (consumer is cold, no relays driven yet).
FAILURES=$(metric_get c1 lava_consumer_majority_baseline_consensus_failure)
MAJORITY=$(metric_get c1 lava_consumer_majority_baseline_value)
CHAIN_HEAD_BEFORE_RELAY=$(metric_get c1 lava_consumer_chain_state_latest_block)
REJECTED_BEFORE=$(metric_get c1 lava_consumer_set_latest_block_outlier_rejected_total)
log "metrics after consensus-failure window: failures=$FAILURES majority_baseline=$MAJORITY chain_head=$CHAIN_HEAD_BEFORE_RELAY rejected=$REJECTED_BEFORE"

# Drive 10 relays — BELT-AND-SUSPENDERS coverage of the relay-path
# SetLatestBlock (relay_processor.go:465). The probe path already landed
# all 3 lies during the 15s sleep above (per the header's "Why all three
# SetLatestBlock paths land lies during this window" section), so
# chain_state.latestBlock should already equal LIE_3. The relays here
# additionally exercise the relay-response writer's SetLatestBlock — a
# code change that breaks ONE of the two paths but not the other would
# be caught by the probe-path landing while assertion (3) still passes;
# adding the relays gives us defense-in-depth observability via the
# CONSUMERS_1.log debug trace ("setting latest block" log lines).
log "driving 10 relays through consumer 1 to additionally exercise the relay-path SetLatestBlock"
for i in $(seq 1 10); do
    curl -s -H 'dapp-id: alice' http://127.0.0.1:${CONSUMER1_TMRPC_PORT}/block > /dev/null || true
    sleep 0.1
done

# --- Phase 3: assertions ---------------------------------------------------

# (1) consensus_failure counter advanced — proves Phase 2b's failure branch fired.
log "lava_consumer_majority_baseline_consensus_failure (sum across apiInterfaces): $FAILURES"
if [ "$FAILURES" -le 0 ]; then
    log "ASSERT FAILED (1): expected consensus_failure to advance, got $FAILURES"
    log "  (possible cause: bucketWidth larger than expected — check --majority-baseline-bucket-time-window flag)"
    exit 1
fi
log "  ✓ (1) consensus_failure counter advanced ($FAILURES failures observed)"

# (2) majority_baseline_value reset to 0 — proves CSM correctly disables the
# IsOutlier guard when consensus fails. Note: this is a SUM across apiInterfaces,
# so we expect 0 even on multi-CSM consumers.
log "lava_consumer_majority_baseline_value (sum across apiInterfaces): $MAJORITY"
if [ "$MAJORITY" -ne 0 ]; then
    log "ASSERT FAILED (2): expected majority_baseline_value=0 under no-consensus, got $MAJORITY"
    log "  (possible cause: one apiInterface DID reach consensus while another didn't — check per-interface breakdown in /metrics)"
    exit 1
fi
log "  ✓ (2) majority_baseline_value reset to 0 (IsOutlier guard disabled)"

# (3) chain_state.latestBlock == LIE_3. The probe path (which fires per
# cycle regardless of user traffic) calls SetLatestBlock for each of the
# 3 lies; the IsOutlier guard is inert (majorityBaseline=0); the monotonic
# check picks the MAX. So with providers reporting (LIE_1, LIE_2, LIE_3),
# chain_state.latestBlock deterministically equals LIE_3 after one full
# probe round. The 15s wait above is 3 probe cycles — way more than
# needed; any drift from LIE_3 indicates a code change to one of the
# SetLatestBlock write sites or the IsOutlier guard's cold-start behavior.
#
# Failure-mode interpretations:
#   - chain_head == 0 / cold-start value: probes are NOT calling SetLatestBlock
#     (something broke at provider_optimizer.go:235/299). Probably a code
#     change that added a new gate.
#   - chain_head < LIE_3 but > 0: some probes landed but not all — possibly
#     a flaky proxy or the optimizer hit an early-return before reaching
#     SetLatestBlock for the highest-block provider. Inspect CONSUMERS_1.log
#     for "setting latest block" lines to see which lies landed.
#   - chain_head > LIE_3: indicates another scenario poisoned chain_state
#     with a higher value and restart_consumer didn't reset it (shouldn't
#     happen — NewChainState starts at 0).
CHAIN_HEAD_AFTER_RELAY=$(metric_get c1 lava_consumer_chain_state_latest_block)
log "lava_consumer_chain_state_latest_block after relays: $CHAIN_HEAD_BEFORE_RELAY → $CHAIN_HEAD_AFTER_RELAY (expected exactly $LIE_3, MAX of $LIE_1/$LIE_2/$LIE_3)"
if [ "$CHAIN_HEAD_AFTER_RELAY" -ne "$LIE_3" ]; then
    log "ASSERT FAILED (3): chain head should equal $LIE_3 (MAX lie locked in by monotonic check), got $CHAIN_HEAD_AFTER_RELAY"
    log "  (see header's 'failure-mode interpretations' for diagnostic guidance)"
    exit 1
fi
log "  ✓ (3) chain_state.latestBlock locked in at $LIE_3 (the MAX lie) — the security-gap behavior is confirmed"

# (4) outlier_rejected counter stayed at 0 — proves the IsOutlier guard was
# inert. This is the CENTRAL documentation: there are NO rejection events
# during a consensus-failure window, so operators must rely on (1)+(2) for
# detection, not on rejection rate.
REJECTED_AFTER=$(metric_get c1 lava_consumer_set_latest_block_outlier_rejected_total)
log "lava_consumer_set_latest_block_outlier_rejected_total: $REJECTED_BEFORE → $REJECTED_AFTER"
if [ "$REJECTED_AFTER" -ne "$REJECTED_BEFORE" ]; then
    log "ASSERT FAILED (4): outlier_rejected counter advanced ($REJECTED_BEFORE → $REJECTED_AFTER) but consensus failure should make IsOutlier inert"
    log "  (possible cause: some apiInterface DID reach consensus during the window — check assertion 2 first)"
    exit 1
fi
log "  ✓ (4) outlier_rejected counter stayed at $REJECTED_BEFORE (IsOutlier guard confirmed inert)"

# --- Phase 4: cleanup -------------------------------------------------------

# Reset proxies but DO NOT restart consumer/providers. This deliberately
# leaves the consumer's chain_state.latestBlock in its poisoned state so
# that A.5 can verify the AlignLatestBlockWithConsensus realignment fires
# on the next consensus cycle. A.5 begins with proxy_reset_all already
# done (idempotent) + restart_provider 1/2/3 to clear chainTrackers, then
# observes the recovery.
log "resetting all proxy overrides (consumer chain_state intentionally left poisoned for A.5)"
proxy_reset_all
log "a4_consensus_failure_window PASS"
