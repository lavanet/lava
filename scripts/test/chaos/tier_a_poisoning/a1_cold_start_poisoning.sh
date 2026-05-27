#!/bin/bash
# scripts/test/chaos/tier_a_poisoning/a1_cold_start_poisoning.sh
#
# Tier A scenario #1: cold-start poisoning.
#
# Validates testing-plan §3.2 row A.2.1: consumer + 3 providers cold-start with
# provider 1 reporting an inflated block height from its very first probe.
# Without the unification's IsOutlier guard + AlignLatestBlockWithConsensus
# recovery (Steps 2 + 4 of the roadmap), the consumer would adopt provider 1's
# lie as the chain head and penalize the honest providers 2+3.
#
# Pass criteria (5 assertions covering the 6 sub-conditions in the plan):
#   (3)   lava_consumer_probe_outlier_blocked advances — Phase 2c invoked blockProvider
#         on the lying provider (counter — "was ever detected as outlier this run")
#   (4)   lava_consumer_align_latest_block_calls_total{outcome="poisoning_reset"}
#         advances — the recovery path fired (lie landed in chainState during
#         cold-start window, then was snapped down once consensus was reached)
#   (1+5) lava_consumer_chain_state_latest_block reflects HONEST consensus, NOT the lie
#   (2)   lava_consumer_majority_baseline_value reflects HONEST consensus
#   (6)   lava_consumer_provider_blocked has a series with value=1 — at least one
#         provider is CURRENTLY isolated from the session pool (gauge — "is right
#         now in the blocked list"; complementary to the counter in assertion 3)
#
# Sequencing note: the lie MUST be injected on proxy 1 BEFORE restart_provider 1
# so provider 1's chainTracker has no prior honest state to fall back on. This
# is what makes the scenario specifically a "cold-start" poisoning test.

set -eu -o pipefail

__dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$__dir"/../lib/common.sh
source "$__dir"/../lib/proxy_control.sh
source "$__dir"/../lib/metrics.sh
source "$__dir"/../lib/assertions.sh

# Fixed injection value rather than dynamic baseline+100k: at cold start the
# consumer has no /metrics view of the chain yet. Local lavad chain advances
# ~1 block/sec and tests rarely run >10min, so chain head stays under 100k.
# 1_000_000 leaves a comfortable margin while being numerically distinct from
# genuine consensus values for human-readable diagnostics.
INJECTED_BLOCK=1000000

# --- Phase 1: cold-start setup --------------------------------------------

log "starting a1_cold_start_poisoning scenario"
setup_env
proxy_reset_all

# Inject the lie BEFORE restarting providers. Provider 1's chainTracker reads
# from proxy 1; the override must be in place before the very first poll so
# the chainTracker has no prior honest state.
log "injecting block override $INJECTED_BLOCK on proxy 1 BEFORE provider restart"
proxy_set_block 1 "$INJECTED_BLOCK"

# Restart all 3 providers per the testing plan spec ("Restart consumer + 3
# providers"). Providers 2 and 3 come up against transparent proxies and report
# honest chain heads; provider 1 comes up against the lying proxy.
log "restarting all 3 providers (cold start)"
restart_provider 1
restart_provider 2
restart_provider 3

# Restart consumer 1 last so its initial probes hit fully-warmed providers.
restart_consumer 1

# Wait long enough to cover the full cold-start poisoning + recovery cycle:
#  * 1 probe cycle (~5s) — first probe carries provider 1's lie into the
#    consumer's optimizer. With majorityBaseline=0 (cold start), IsOutlier
#    returns false; the lie lands in chainState.latestBlock.
#  * Phase 2b on the same / next cycle — consensus is computed from providers
#    2 + 3 majority; majorityBaseline is set to the honest floor; the gap
#    between chainState.latestBlock (lie) and floor exceeds threshold;
#    AlignLatestBlockWithConsensus fires the poisoning_reset path.
#  * 1 more cycle for metrics to settle.
# 15s is the safe over-estimate.
log "waiting 15s for cold-start probe cycles + consensus + realignment"
sleep 15

# --- Phase 2: assertions --------------------------------------------------

# (3) Outlier-blocking detection: counter must be > 0. Cold-start consumer
# starts at 0, so any positive value proves Phase 2c blocked the lying provider.
OUTLIER_BLOCKED=$(metric_get c1 lava_consumer_probe_outlier_blocked)
log "lava_consumer_probe_outlier_blocked total: $OUTLIER_BLOCKED"
if [ "$OUTLIER_BLOCKED" -le 0 ]; then
    log "ASSERT FAILED (3): expected outlier provider to be blocked, got 0"
    exit 1
fi
log "  ✓ (3) outlier provider was blocked (counter=$OUTLIER_BLOCKED)"

# (4) Recovery path: lava_consumer_align_latest_block_calls_total with
# outcome=poisoning_reset must have fired at least once. This is the proof
# that the lie made it into chainState.latestBlock and was then realigned
# down by consensus — i.e. the full Step 2 + Step 4 recovery worked.
#
# Label filter form: pass `outcome="poisoning_reset"` (no surrounding braces)
# so metric_get's substring grep matches the label inside multi-label series
# like `{apiInterface="rest",outcome="poisoning_reset",spec="LAV1"}`. Wrapping
# in `{...}` would require those to be the metric's ONLY labels.
POISONING_RESETS=$(metric_get c1 lava_consumer_align_latest_block_calls_total 'outcome="poisoning_reset"')
log "lava_consumer_align_latest_block_calls_total with outcome=poisoning_reset (sum across apiInterfaces): $POISONING_RESETS"
if [ "$POISONING_RESETS" -le 0 ]; then
    log "ASSERT FAILED (4): expected poisoning_reset realignment to fire at least once, got 0"
    exit 1
fi
log "  ✓ (4) poisoning_reset realignment path fired ($POISONING_RESETS times)"

# (1+5) chainState.latestBlock holds the HONEST consensus value. We don't know
# the exact chain head, but it must be in the realistic local-chain range and
# nowhere near the injected lie ($INJECTED_BLOCK).
CHAIN_HEAD=$(metric_get c1 lava_consumer_chain_state_latest_block)
log "lava_consumer_chain_state_latest_block: $CHAIN_HEAD (lie was $INJECTED_BLOCK)"
if [ "$CHAIN_HEAD" -le 0 ] || [ "$CHAIN_HEAD" -ge 100000 ]; then
    log "ASSERT FAILED (1+5): chain head should be honest (0 < x < 100000), got $CHAIN_HEAD"
    exit 1
fi
log "  ✓ (1+5) chainState.latestBlock reflects honest consensus, NOT the lie"

# (2) majorityBaseline value reflects honest consensus. Sum across 3
# apiInterfaces, so divide an upper bound of 100000 by 3 → 300000 ceiling.
MAJORITY=$(metric_get c1 lava_consumer_majority_baseline_value)
log "lava_consumer_majority_baseline_value (sum across apiInterfaces): $MAJORITY"
if [ "$MAJORITY" -le 0 ] || [ "$MAJORITY" -ge 300000 ]; then
    log "ASSERT FAILED (2): majority_baseline_value should be honest sum (0 < x < 300000), got $MAJORITY"
    exit 1
fi
log "  ✓ (2) majority_baseline_value reflects honest consensus"

# (6) At least one provider is currently in the blocked list. The gauge
# `lava_consumer_provider_blocked` is 0 or 1 per (provider, apiInterface)
# series; summing across all series yields the total count of currently-blocked
# (provider, apiInterface) pairs. With the lie still active on proxy 1, the
# lying provider should be blocked across all 3 apiInterfaces → expect sum ≥ 1
# (typically 3). Counter assertion 3 already proved blockProvider() was CALLED;
# this gauge assertion proves the consumer is CURRENTLY isolating the provider.
BLOCKED_COUNT=$(metric_get c1 lava_consumer_provider_blocked)
log "lava_consumer_provider_blocked (sum, # of currently-blocked provider-apiInterface pairs): $BLOCKED_COUNT"
if [ "$BLOCKED_COUNT" -lt 1 ]; then
    log "ASSERT FAILED (6): expected at least one provider currently in blocked list, got $BLOCKED_COUNT"
    exit 1
fi
log "  ✓ (6) at least one provider is currently isolated from the session pool"

# --- Phase 3: cleanup -----------------------------------------------------

log "resetting all proxy overrides"
proxy_reset_all
log "a1_cold_start_poisoning PASS"
