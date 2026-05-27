#!/bin/bash
# scripts/test/chaos/tier_a_poisoning/a3_cache_poisoning.sh
#
# Tier A scenario #3: cache poisoning via shared-state.
#
# Validates testing-plan §3.2 row A.2.3. Differs from A.1/A.2 in what's
# attacked: A.1/A.2 corrupt a provider's reported chain head (the relay/probe
# path). A.3 corrupts the shared-state cache directly (the cross-consumer
# coherence path). Both honest providers and the local chainTracker are
# untouched — the attack vector is a forged entry on the chain-wide
# `latestBlockKey` in the shared cache that one consumer reads on the next
# relay's cache GET.
#
# Production analogue: an attacker with write access to the shared cache
# backend (or a misbehaving co-consumer in a sharded deployment) injects a
# very high seenBlock to make all other consumers adopt the lie as the chain
# head — driving them to penalize honest providers and trust a future cache
# entry that promises the impossible.
#
# The unification's defense: `SetLatestBlockFromSharedState` (chain_state.go:208)
# wraps `SetLatestBlock`, which runs through the IsOutlier guard. With
# majorityBaseline armed (consumer 2 is warm), the forged 1,000,000 value is
# rejected at the front door — M7 (propagation attempted) and M2 (outlier
# rejected) both fire, while M3 (chain_state_latest_block) stays honest.
#
# Why a Go injector (scripts/test/chaos/tools/cache_poison) instead of
# grpcurl: the cache's proto chain transitively imports cosmos-sdk types that
# are non-trivial to assemble for grpcurl --proto. The Go tool re-uses the
# project's already-generated `pairingtypes` Go types so wire format is
# byte-identical to a real consumer's SetRelay. See the tool's package
# doc-comment for full rationale.
#
# Pass criteria (3 assertions covering the testing-plan sub-conditions):
#   (1)   consumer 2's lava_consumer_shared_state_propagations_total advances —
#         proves the consumer's cache-GET path saw the poisoned chain-wide
#         seenBlock and ATTEMPTED the propagation (M7 fires unconditionally
#         inside SetLatestBlockFromSharedState).
#   (2)   consumer 2's lava_consumer_set_latest_block_outlier_rejected_total
#         advances — proves the IsOutlier guard rejected the forged value at
#         the front door (the M2 emit inside SetLatestBlock).
#   (3)   consumer 2's lava_consumer_chain_state_latest_block stayed honest —
#         end-to-end proof that the lie never reached the unified tracker.

set -eu -o pipefail

__dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$__dir"/../lib/common.sh
source "$__dir"/../lib/proxy_control.sh
source "$__dir"/../lib/metrics.sh
source "$__dir"/../lib/assertions.sh

# Same fixed injection value as A.1/A.2 — well above any plausible local-chain
# head, numerically distinct from genuine consensus for diagnostics.
INJECTED_BLOCK=1000000

# Cache TTL for non-finalized entries is 500ms (cache.DefaultExpirationForNonFinalized).
# We inject + curl repeatedly to ensure at least one consumer-2 relay lands
# inside a poison window.
INJECT_CYCLES=20
INJECT_INTERVAL_S=0.2   # 5 injections per second — well under the 500ms TTL.

CHAIN_ID=LAV1
CACHE_ADDR="127.0.0.1:${CACHE_LISTEN_PORT}"
POISON_BIN=/tmp/cache_poison

# --- Phase 0: build the injector tool ---------------------------------------

# Build once up-front. `go run` per iteration would add ~300ms × INJECT_CYCLES
# of compile overhead — far longer than the 500ms TTL window — and make it
# essentially impossible for any relay to see a poisoned cache entry.
log "starting a3_cache_poisoning scenario"
log "building cache_poison tool → $POISON_BIN"
( cd "$REPO_ROOT" && go build -o "$POISON_BIN" ./scripts/test/chaos/tools/cache_poison ) || {
    log "ABORT: failed to build cache_poison tool"
    exit 2
}

# --- Phase 1: warm setup with HONEST providers + consumers ------------------

setup_env
proxy_reset_all

# Clean state on both consumers — A.3 asserts on consumer 2, but consumer 1
# shares the same cache backend so any prior poisoning could leave stale state
# in either's blocked-providers map. Cheaper to restart both than to debug.
restart_provider 1
restart_provider 2
restart_provider 3
restart_consumer 1
restart_consumer 2

# Wait for consumer 2 to reach warm steady-state (majorityBaseline armed).
# Without this, IsOutlier returns false and the scenario's central proof
# point (M2 firing on rejection) won't fire — instead the lie would land
# in chainState.latestBlock and we'd be testing the cold-start recovery
# path (which is A.1's job, not A.3's).
log "waiting 15s for consumer 2 to reach warm steady-state with honest providers"
sleep 15

# Sanity check on consumer 2's starting state.
MAJORITY_C2=$(metric_get c2 lava_consumer_majority_baseline_value)
if [ "$MAJORITY_C2" -le 0 ]; then
    log "ABORT: consumer 2 majority_baseline_value=0 — IsOutlier guard inert; scenario can't validate as intended"
    exit 2
fi
log "  baseline majority_baseline_value on c2 (sum): $MAJORITY_C2 (IsOutlier guard armed)"

# --- Phase 2: capture pre-poisoning baselines on consumer 2 -----------------

PROP_BASELINE=$(metric_get c2 lava_consumer_shared_state_propagations_total)
REJECTED_BASELINE=$(metric_get c2 lava_consumer_set_latest_block_outlier_rejected_total)
CHAIN_HEAD_BASELINE=$(metric_get c2 lava_consumer_chain_state_latest_block)
log "baselines on c2: propagations=$PROP_BASELINE rejected=$REJECTED_BASELINE chain_head=$CHAIN_HEAD_BASELINE"

# --- Phase 3: inject + drive relays in a tight loop -------------------------

# The interleaving (inject → curl → inject → curl ...) gives every curl the
# best chance of hitting a live poison window. We exit the loop the moment
# BOTH M7 and M2 have advanced — no point continuing to attack a consumer
# that's already proven its defenses.
log "interleaving cache injections + consumer 2 relays ($INJECT_CYCLES cycles, ${INJECT_INTERVAL_S}s apart)"
INJECTIONS_DONE=0
RELAYS_SENT=0
for i in $(seq 1 $INJECT_CYCLES); do
    # Inject the lie into the chain-wide key. Tool exits non-zero on failure;
    # don't abort the whole scenario on a single failed injection — could be
    # a transient gRPC blip.
    "$POISON_BIN" "$CACHE_ADDR" "$CHAIN_ID" "$INJECTED_BLOCK" >/dev/null 2>&1 || {
        log "  WARN: cache_poison injection $i failed (continuing)"
        continue
    }
    INJECTIONS_DONE=$((INJECTIONS_DONE + 1))

    # Fire a relay through consumer 2. This triggers cache.GetEntry — the
    # consumer reads back our poisoned chain-wide key → calls
    # SetLatestBlockFromSharedState(1000000) → M7 + M2 emit.
    curl -s -H 'dapp-id: alice' "http://127.0.0.1:${CONSUMER2_TMRPC_PORT}/block" > /dev/null || true
    RELAYS_SENT=$((RELAYS_SENT + 1))

    # Early-exit when both proof-points have advanced.
    CUR_PROP=$(metric_get c2 lava_consumer_shared_state_propagations_total)
    CUR_REJECTED=$(metric_get c2 lava_consumer_set_latest_block_outlier_rejected_total)
    if [ "$CUR_PROP" -gt "$PROP_BASELINE" ] && [ "$CUR_REJECTED" -gt "$REJECTED_BASELINE" ]; then
        log "  both metrics advanced after $INJECTIONS_DONE injections / $RELAYS_SENT relays"
        log "    propagations: $PROP_BASELINE → $CUR_PROP"
        log "    rejected: $REJECTED_BASELINE → $CUR_REJECTED"
        break
    fi
    sleep "$INJECT_INTERVAL_S"
done

# --- Phase 4: assertions ----------------------------------------------------

# (1) M7 advanced — cache-GET path saw the poisoned chain-wide seenBlock and
# called SetLatestBlockFromSharedState. The emit is unconditional inside that
# wrapper, so any successful read of the poisoned key fires this.
PROP_CURRENT=$(metric_get c2 lava_consumer_shared_state_propagations_total)
log "lava_consumer_shared_state_propagations_total on c2: $PROP_BASELINE → $PROP_CURRENT"
if [ "$PROP_CURRENT" -le "$PROP_BASELINE" ]; then
    log "ASSERT FAILED (1): expected shared_state_propagations_total to advance, stayed at $PROP_BASELINE"
    log "  (possible cause: cache TTL expired between every inject + curl pair; try lowering INJECT_INTERVAL_S)"
    exit 1
fi
log "  ✓ (1) consumer 2's cache-GET saw the poisoned chain-wide seenBlock"

# (2) M2 advanced — the IsOutlier guard caught the forged value. This is the
# unification's central proof point for A.3.
REJECTED_CURRENT=$(metric_get c2 lava_consumer_set_latest_block_outlier_rejected_total)
log "lava_consumer_set_latest_block_outlier_rejected_total on c2: $REJECTED_BASELINE → $REJECTED_CURRENT"
if [ "$REJECTED_CURRENT" -le "$REJECTED_BASELINE" ]; then
    log "ASSERT FAILED (2): expected set_latest_block_outlier_rejected_total to advance, stayed at $REJECTED_BASELINE"
    log "  (possible cause: consumer 2 wasn't actually warm — re-check baseline majority_baseline_value)"
    exit 1
fi
log "  ✓ (2) IsOutlier guard rejected the forged seenBlock at the front door"

# (3) ChainState.latestBlock stayed honest. End-to-end proof that the lie
# never reached the unified tracker. Same upper-bound rationale as A.1/A.2:
# local chain advances ~1 block/sec and tests rarely run >10min, so any
# realistic head stays well under 100k. The injected lie is 1_000_000.
CHAIN_HEAD_CURRENT=$(metric_get c2 lava_consumer_chain_state_latest_block)
log "lava_consumer_chain_state_latest_block on c2: $CHAIN_HEAD_BASELINE → $CHAIN_HEAD_CURRENT (lie was $INJECTED_BLOCK)"
if [ "$CHAIN_HEAD_CURRENT" -le 0 ] || [ "$CHAIN_HEAD_CURRENT" -ge 100000 ]; then
    log "ASSERT FAILED (3): chain head should stay honest (0 < x < 100000), got $CHAIN_HEAD_CURRENT"
    exit 1
fi
if [ "$CHAIN_HEAD_CURRENT" -lt "$CHAIN_HEAD_BASELINE" ]; then
    log "ASSERT FAILED (3): chain head went DOWN ($CHAIN_HEAD_BASELINE → $CHAIN_HEAD_CURRENT); should advance with chain progress"
    exit 1
fi
log "  ✓ (3) consumer 2's chainState.latestBlock stayed honest, advanced naturally with chain"

# --- Phase 5: cleanup -------------------------------------------------------

# The cache entry will TTL out within 500ms of the last injection, so no
# explicit cache cleanup is needed. proxy_reset_all is here for parity with
# A.1/A.2 in case future edits add proxy-side state.
log "resetting all proxy overrides"
proxy_reset_all
log "a3_cache_poisoning PASS"
