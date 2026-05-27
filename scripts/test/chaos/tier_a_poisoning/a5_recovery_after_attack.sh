#!/bin/bash
# scripts/test/chaos/tier_a_poisoning/a5_recovery_after_attack.sh
#
# Tier A scenario #5: recovery after consensus-failure poisoning.
#
# Validates testing-plan §3.2 row A.2.5. Chains conceptually off A.4: the
# scenario A.4 documents (the consensus-failure security gap where the MAX
# provider lie ends up locked in `chain_state.latestBlock` via the monotonic
# check) is the SETUP for A.5. A.5 then proves the recovery mechanism —
# `AlignLatestBlockWithConsensus` with `outcome="poisoning_reset"` — works
# as designed: it snaps `chain_state.latestBlock` DOWN to the honest floor
# on the first probe cycle after consensus is restored. This is the ONLY
# code path that can move the unified tracker downward; SetLatestBlock is
# monotonic by design.
#
# Why self-contained instead of relying on A.4's exit state: scenarios should
# be runnable independently for debug iteration. A.5 reproduces A.4's setup
# internally (the operations are idempotent — proxy_set_block + restart
# work regardless of prior state). When A.5 runs immediately after A.4 (as
# in run_all.sh's alphabetical iteration), the setup phase is harmless
# repetition; when A.5 runs standalone, it works correctly from scratch.
#
# Production analogue: a network partition resolved, a buggy provider
# release rolled back, or a chain reorg settled — in all cases the system
# must converge back to honest consensus without operator intervention,
# AND must do so without permanently penalizing providers whose probes
# happened to fall on the wrong side during the failure window.
#
# --- Recovery sequence (what we're testing) ---------------------------------
#
# State at recovery start (carrying over from poisoning bootstrap):
#   - chain_state.latestBlock = 130000 (locked in by monotonic from A.4's lies)
#   - majority_baseline_value = 0 (consensus was failing)
#   - 3 provider chainTrackers locked at (10000, 60000, 130000)
#   - proxy_reset_all has cleared the proxy overrides
#
# Then:
#   1. restart_provider 1/2/3 — clears chainTrackers; new chainTrackers read
#      transparent proxies → all report real chain head (~85, the local dev chain)
#   2. Next probe cycle (~5s after consumer's next interval): probes hit all
#      3 providers → results [LatestBlock=85, 85, 85] arrive
#   3. Phase 2a: optimizer's AppendProbeRelayData runs. IsOutlier(85) at this
#      moment: majorityBaseline is still 0 → returns false → SetLatestBlock(85)
#      called → monotonic check: 85 <= 130000 → REJECTED (no-op). The lie
#      stays locked in until step 4 fires.
#   4. Phase 2b: ComputeMajorityBaseline([85, 85, 85], 120) → all in same
#      bucket (windowWidth=240, well > 0 spread) → bestCount=3 >= majority=2
#      → returns 85. CSM enters the SUCCESS branch:
#      a. SetMajorityBaseline(85, threshold=100, apiInterface) — M4 advances
#         from 0 to 85 per apiInterface
#      b. AlignLatestBlockWithConsensus(85, 100, apiInterface) — the key
#         recovery call. chain_state.go:148-197 trace:
#         - floor=85 > 0 → don't return at line 154
#         - cs.latestBlock=130000 > floor=85 → don't return at line 160
#         - gap = 130000 - 85 = 129915
#         - gap > threshold=100 → poisoning_reset branch (line 170)
#         - M1: SetAlignLatestBlockOutcome(chainID, apiInterface, "poisoning_reset")
#         - M5: SetAlignLatestBlockGap(chainID, apiInterface, 129915)
#         - cs.latestBlock = 85   ← the actual realignment write
#         - M3: SetChainStateLatestBlock(chainID, 85)
#      c. blockOutlierProviders runs with floor=85, threshold=100. Each
#         provider reports LatestBlock=85, gap=0 < 100 → no blocking.
#         provider_blocked stays at 0.
#   5. Subsequent probe cycles: latestBlock=85 == floor=85 → healthy_no_change
#      path → counter ticks healthy_no_change instead of poisoning_reset.
#
# --- Per-apiInterface accounting -------------------------------------------
#
# The consumer runs 4 apiInterface goroutines (rest + tendermintrpc +
# tendermintrpc-uri + grpc — see §5.A.4 discovery #5). Each runs its own
# Phase 2b, so AlignLatestBlockWithConsensus is called up to 4 times in
# quick succession on the recovery cycle. However, ChainState's mutex
# serializes them:
#   - First caller: sees latestBlock=130000 > floor=85 → fires poisoning_reset,
#     sets cs.latestBlock=85
#   - Subsequent callers: see latestBlock=85 <= floor=85 → fire healthy_no_change
# So `poisoning_reset` counter advances by exactly 1 (modulo race timing
# nuances; the assertion uses `> 0` to be robust).
#
# --- Pass criteria (5 assertions, A.2.5 sub-conditions) --------------------
#
#   (1)   `lava_consumer_chain_state_latest_block` < 100000 — proves the
#         realignment WRITE happened (cs.latestBlock = floor at chain_state.go:193).
#         Strictly: should be ~85 (the honest chain head); the < 100000 bound
#         leaves natural-chain-advance headroom.
#   (2)   `lava_consumer_majority_baseline_value` > 0 — proves consensus is
#         restored. Sum across apiInterfaces; expect ~85 × 4 ≈ 340 (matches
#         A.3's observation of 240 when warm).
#   (3)   `lava_consumer_align_latest_block_calls_total{outcome="poisoning_reset"}`
#         advanced past pre-recovery baseline (which is 0 — bootstrap restart
#         zeroed it) — proves the SPECIFIC recovery code path fired, not
#         healthy_no_change or revert.
#   (4)   `lava_consumer_align_latest_block_last_gap_blocks` > 1000 — proves
#         the realignment MAGNITUDE was significant. Expect ~129915.
#   (5)   `lava_consumer_provider_blocked` == 0 — defensive: proves the
#         recovery didn't falsely block any provider. With honest values
#         post-restart, blockOutlierProviders has nothing to block.

set -eu -o pipefail

__dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$__dir"/../lib/common.sh
source "$__dir"/../lib/proxy_control.sh
source "$__dir"/../lib/metrics.sh
source "$__dir"/../lib/assertions.sh

# Same values as A.4 — keeps the two scenarios numerically linked for
# diagnostic correlation (a chain_state.latestBlock=130000 in either A.4
# or A.5 logs unambiguously means "MAX lie landed").
LIE_1=10000
LIE_2=60000
LIE_3=130000

# --- Phase 1: bootstrap the A.4 poisoning state -----------------------------
#
# We reproduce A.4's setup so A.5 can be run standalone. If A.4 just ran,
# this is harmless: proxy_set_block + restart_provider are idempotent.

log "starting a5_recovery_after_attack scenario"
setup_env
proxy_reset_all

log "Phase 1: reproducing A.4 poisoning state (proxies $LIE_1 / $LIE_2 / $LIE_3)"
proxy_set_block 1 "$LIE_1"
proxy_set_block 2 "$LIE_2"
proxy_set_block 3 "$LIE_3"
restart_provider 1
restart_provider 2
restart_provider 3
restart_consumer 1

# Wait long enough for the consumer to reach "fully poisoned" state:
#   - 3 probe cycles for consensus_failure to fire multiple times
#   - Probe-path SetLatestBlock to land all 3 lies (MAX=130000 locks in)
log "waiting 15s for consumer to reach the poisoned state (chain_state.latestBlock should reach $LIE_3)"
sleep 15

# Pre-condition sanity check: scenario only validates if we're actually in
# the poisoned state at this point. If not, something broke in bootstrap.
POISONED_HEAD=$(metric_get c1 lava_consumer_chain_state_latest_block)
POISONED_MAJORITY=$(metric_get c1 lava_consumer_majority_baseline_value)
if [ "$POISONED_HEAD" -ne "$LIE_3" ]; then
    log "ABORT: pre-recovery chain_state.latestBlock should be $LIE_3, got $POISONED_HEAD"
    log "  (A.4 should have demonstrated this state — re-run A.4 first to confirm bootstrap works)"
    exit 2
fi
if [ "$POISONED_MAJORITY" -ne 0 ]; then
    log "ABORT: pre-recovery majority_baseline_value should be 0 (consensus failing), got $POISONED_MAJORITY"
    exit 2
fi
log "  ✓ pre-recovery state confirmed: chain_state=$POISONED_HEAD, majority=$POISONED_MAJORITY"

# Capture baselines for assertions (3) + (4). Both should be 0 post-restart_consumer.
POISONING_RESET_BASELINE=$(metric_get c1 lava_consumer_align_latest_block_calls_total 'outcome="poisoning_reset"')
LAST_GAP_BASELINE=$(metric_get c1 lava_consumer_align_latest_block_last_gap_blocks)
log "  align baselines: poisoning_reset=$POISONING_RESET_BASELINE, last_gap=$LAST_GAP_BASELINE"

# --- Phase 2: restore healthy providers (the recovery trigger) --------------
#
# Two-step restore:
#   1. proxy_reset_all clears the override fields so subsequent proxy reads
#      pass through to the real lavad. (Idempotent — A.4 already did this,
#      but we re-call here in case A.5 ran standalone.)
#   2. restart_provider clears each provider's monotonic chainTracker. New
#      chainTrackers read the now-transparent proxies → observe real chain head.

log "Phase 2: restoring healthy providers"
proxy_reset_all
restart_provider 1
restart_provider 2
restart_provider 3

# DELIBERATELY no restart_consumer here — we want consumer 1's
# chain_state.latestBlock=130000 to persist into the recovery cycle so that
# the realignment write at chain_state.go:193 can be observed.

# Wait for at least 2 probe cycles after providers come up healthy:
#   - First cycle: providers respond honestly, Phase 2b succeeds with floor≈85,
#     AlignLatestBlockWithConsensus fires poisoning_reset
#   - Second cycle: cs.latestBlock=85 == floor=85, healthy_no_change fires
# 15s buffers against probe-cycle timing alignment uncertainty.
log "waiting 15s for ~2-3 healthy probe cycles to complete the recovery"
sleep 15

# --- Phase 3: assertions ---------------------------------------------------

# (1) chain_state.latestBlock realigned DOWN. Was $LIE_3 (130000); should
# now be ~85 (the honest chain head). The < 100000 bound is loose but
# unambiguous — anything in that range means the lie was overwritten by
# the realignment, not just rejected by some other mechanism.
RECOVERED_HEAD=$(metric_get c1 lava_consumer_chain_state_latest_block)
log "lava_consumer_chain_state_latest_block: $POISONED_HEAD → $RECOVERED_HEAD (honest expected ~85)"
if [ "$RECOVERED_HEAD" -le 0 ] || [ "$RECOVERED_HEAD" -ge 100000 ]; then
    log "ASSERT FAILED (1): chain_state should be realigned to honest range (0 < x < 100000), got $RECOVERED_HEAD"
    log "  (interpretation: the recovery write at chain_state.go:193 did NOT execute — investigate AlignLatestBlockWithConsensus call site at consumer_session_manager.go:539)"
    exit 1
fi
log "  ✓ (1) chain_state.latestBlock realigned to honest range — recovery write fired"

# (2) majority_baseline_value advanced. Was 0; should now be ~85 per apiInterface
# (sum across 4 apiInterfaces ≈ 340).
RECOVERED_MAJORITY=$(metric_get c1 lava_consumer_majority_baseline_value)
log "lava_consumer_majority_baseline_value (sum): $POISONED_MAJORITY → $RECOVERED_MAJORITY"
if [ "$RECOVERED_MAJORITY" -le 0 ]; then
    log "ASSERT FAILED (2): majority_baseline_value should advance from 0, got $RECOVERED_MAJORITY"
    log "  (interpretation: Phase 2b's success branch never fired — check ComputeMajorityBaseline; providers may not be reporting honest values yet)"
    exit 1
fi
log "  ✓ (2) majority_baseline_value advanced — consensus is restored"

# (3) poisoning_reset counter advanced. This is the CENTRAL proof of A.5:
# the specific recovery path (chain_state.go:170-177) fired.
POISONING_RESET_CURRENT=$(metric_get c1 lava_consumer_align_latest_block_calls_total 'outcome="poisoning_reset"')
log "lava_consumer_align_latest_block_calls_total{outcome=poisoning_reset}: $POISONING_RESET_BASELINE → $POISONING_RESET_CURRENT"
if [ "$POISONING_RESET_CURRENT" -le "$POISONING_RESET_BASELINE" ]; then
    log "ASSERT FAILED (3): poisoning_reset should advance during recovery, stayed at $POISONING_RESET_BASELINE"
    log "  (interpretation: the recovery path took a different branch — possibly healthy_no_change if the lie wasn't actually high enough, or revert if gap was small. Check CONSUMERS_1.log for 'latestBlock reset' lines.)"
    exit 1
fi
log "  ✓ (3) poisoning_reset path fired — specific recovery code path confirmed"

# (4) last_gap_blocks reflects the realignment magnitude. We expect ~129915
# (= 130000 - ~85). The > 1000 bound is loose; tightening to ~129915 would
# be fragile against natural chain advance during the test window. 1000 is
# unambiguously "much larger than the threshold (100)" so we know the gauge
# was set by the poisoning_reset branch, not by an incidental revert.
LAST_GAP_CURRENT=$(metric_get c1 lava_consumer_align_latest_block_last_gap_blocks)
log "lava_consumer_align_latest_block_last_gap_blocks (sum): $LAST_GAP_BASELINE → $LAST_GAP_CURRENT (expected ~129915)"
if [ "$LAST_GAP_CURRENT" -le 1000 ]; then
    log "ASSERT FAILED (4): last_gap_blocks should reflect a large realignment (> 1000), got $LAST_GAP_CURRENT"
    exit 1
fi
log "  ✓ (4) last_gap_blocks reflects the realignment magnitude — significant downward snap"

# (5) provider_blocked stays at 0. Defensive: verifies the recovery cycle's
# blockOutlierProviders pass (now that floor > 0 and the gate is open) did
# NOT block any provider. With all 3 providers reporting honest ~85,
# nothing exceeds floor+threshold=185, so no blocking.
BLOCKED_AFTER=$(metric_get c1 lava_consumer_provider_blocked)
log "lava_consumer_provider_blocked (sum): $BLOCKED_AFTER"
if [ "$BLOCKED_AFTER" -ne 0 ]; then
    log "ASSERT FAILED (5): no provider should be blocked after recovery (honest values < floor+threshold), got $BLOCKED_AFTER"
    log "  (interpretation: a provider's chainTracker still has stale high values from bootstrap — restart_provider may have failed to clear it)"
    exit 1
fi
log "  ✓ (5) provider_blocked stayed at 0 — recovery did not falsely block any provider"

# --- Phase 4: cleanup -------------------------------------------------------

# Proxies already reset in Phase 2; restart leaves env in clean state for
# subsequent scenarios (the consumer has now reached honest steady-state).
log "a5_recovery_after_attack PASS"
