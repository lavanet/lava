package lavasession

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

// whitelistAllowingAllExcept builds a loaded whitelist that permits every provider produced by
// createPairingList for the given chain and region, except the one excluded address.
func whitelistAllowingAllExcept(t *testing.T, chainID, region, excluded string) *ProviderWhitelist {
	t.Helper()
	addrs := make([]string, 0, numberOfProviders)
	for p := 0; p < numberOfProviders; p++ {
		addr := providerStr + strconv.Itoa(p)
		if addr == excluded {
			continue
		}
		addrs = append(addrs, strconv.Quote(addr))
	}
	pw := NewProviderWhitelist()
	doc := fmt.Sprintf(`{"chains":{%q:{%q:[%s]}}}`, chainID, region, strings.Join(addrs, ","))
	require.NoError(t, pw.UpdateFromBytes([]byte(doc)))
	return pw
}

// TestProviderWhitelist_FilterHelpers covers the building blocks deterministically: the list
// filter drops only non-whitelisted addresses, and passthrough holds when unset/unloaded.
func TestProviderWhitelist_FilterHelpers(t *testing.T) {
	csm := CreateConsumerSessionManager()
	csm.rpcEndpoint.Geolocation = geoEU
	all := []string{providerStr + "0", providerStr + "1", providerStr + "2"}

	// No whitelist configured -> passthrough (input unchanged, everything allowed).
	require.Nil(t, csm.providerWhitelist)
	require.Equal(t, all, csm.filterAllowedProviders(all))
	require.True(t, csm.isProviderAllowed(providerStr+"0"))

	// Whitelist excluding provider0 -> filtered out of the list and individually disallowed.
	excluded := providerStr + "0"
	csm.SetProviderWhitelist(whitelistAllowingAllExcept(t, csm.rpcEndpoint.ChainID, "EU", excluded))
	require.Equal(t, []string{providerStr + "1", providerStr + "2"}, csm.filterAllowedProviders(all))
	require.False(t, csm.isProviderAllowed(excluded))
	require.True(t, csm.isProviderAllowed(providerStr+"1"))
}

// TestProviderWhitelist_FilterByConsumerGeolocation asserts the list filters by the consumer's
// region: a provider listed only for another region is dropped, exactly as if it weren't listed.
// This is the core MAG-1987 behavior for a per-region gateway.
func TestProviderWhitelist_FilterByConsumerGeolocation(t *testing.T) {
	csm := CreateConsumerSessionManager()
	csm.rpcEndpoint.Geolocation = geoEU // this gateway is an EU consumer
	all := []string{providerStr + "0", providerStr + "1"}

	// provider0 whitelisted for EU (the consumer's region), provider1 only for AS.
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(fmt.Sprintf(
		`{"chains":{%q:{"EU":[%q],"AS":[%q]}}}`, csm.rpcEndpoint.ChainID, providerStr+"0", providerStr+"1"))))
	csm.SetProviderWhitelist(pw)

	require.Equal(t, []string{providerStr + "0"}, csm.filterAllowedProviders(all))
	require.True(t, csm.isProviderAllowed(providerStr+"0"))
	require.False(t, csm.isProviderAllowed(providerStr+"1")) // AS-only -> not for this EU consumer
}

// TestProviderWhitelist_FilterOptimizerPath asserts the optimizer never returns a non-whitelisted
// provider (the main selection path, line ~1086).
func TestProviderWhitelist_FilterOptimizerPath(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	csm.rpcEndpoint.Geolocation = geoEU
	require.NoError(t, csm.UpdateAllProviders(firstEpochHeight, createPairingList("", true), nil))
	excluded := providerStr + "0"
	csm.SetProviderWhitelist(whitelistAllowingAllExcept(t, csm.rpcEndpoint.ChainID, "EU", excluded))

	csm.lock.RLock()
	defer csm.lock.RUnlock()
	for i := 0; i < 100; i++ {
		addrs, err := csm.getValidProviderAddresses(ctx, 1, map[string]struct{}{}, cuForFirstRequest, servicedBlockNumber, "", nil, common.NO_STATE, "", "")
		require.NoError(t, err)
		require.NotEmpty(t, addrs)
		require.NotContains(t, addrs, excluded)
	}
}

// TestProviderWhitelist_FilterHeaderSelect asserts a header-selected provider must be whitelisted
// (whitelist wins over the per-request provider hint).
func TestProviderWhitelist_FilterHeaderSelect(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	csm.rpcEndpoint.Geolocation = geoEU
	require.NoError(t, csm.UpdateAllProviders(firstEpochHeight, createPairingList("", true), nil))
	excluded := providerStr + "0"
	allowed := providerStr + "1"
	csm.SetProviderWhitelist(whitelistAllowingAllExcept(t, csm.rpcEndpoint.ChainID, "EU", excluded))

	csm.lock.RLock()
	defer csm.lock.RUnlock()
	// Selecting the non-whitelisted provider via header fails.
	_, err := csm.getValidProviderAddresses(ctx, 1, map[string]struct{}{}, cuForFirstRequest, servicedBlockNumber, "", nil, common.NO_STATE, "", excluded)
	require.Error(t, err)
	// Selecting a whitelisted provider via header returns exactly it.
	addrs, err := csm.getValidProviderAddresses(ctx, 1, map[string]struct{}{}, cuForFirstRequest, servicedBlockNumber, "", nil, common.NO_STATE, "", allowed)
	require.NoError(t, err)
	require.Equal(t, []string{allowed}, addrs)
}

// TestProviderWhitelist_FilterStickySession asserts a sticky session pinned to a now-non-whitelisted
// provider does not relay to it (the sticky shortcut, line ~1116).
func TestProviderWhitelist_FilterStickySession(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	csm.rpcEndpoint.Geolocation = geoEU
	require.NoError(t, csm.UpdateAllProviders(firstEpochHeight, createPairingList("", true), nil))
	excluded := providerStr + "0"
	csm.SetProviderWhitelist(whitelistAllowingAllExcept(t, csm.rpcEndpoint.ChainID, "EU", excluded))

	// Pin a sticky session to the excluded provider.
	csm.stickySessions.Set("sticky-id", &StickySession{Provider: excluded, Epoch: csm.atomicReadCurrentEpoch()})

	csm.lock.RLock()
	defer csm.lock.RUnlock()
	addrs, err := csm.getValidProviderAddresses(ctx, 1, map[string]struct{}{}, cuForFirstRequest, servicedBlockNumber, "", nil, common.NO_STATE, "sticky-id", "")
	require.NoError(t, err)
	require.NotContains(t, addrs, excluded)
}

// TestProviderWhitelist_FilterBlockedRecovery asserts the blocked-provider recovery fallback never
// returns a non-whitelisted provider, even though it was whitelisted when it got blocked (the
// whitelist can change via the hourly refresh; line ~1290).
func TestProviderWhitelist_FilterBlockedRecovery(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	csm.rpcEndpoint.Geolocation = geoEU
	require.NoError(t, csm.UpdateAllProviders(firstEpochHeight, createPairingList("", true), nil))
	excluded := providerStr + "0"
	allowed := providerStr + "5"
	csm.SetProviderWhitelist(whitelistAllowingAllExcept(t, csm.rpcEndpoint.ChainID, "EU", excluded))

	// Simulate both providers being blocked this epoch (excluded was delisted after being blocked).
	csm.lock.Lock()
	csm.currentlyBlockedProviderAddresses = []string{excluded, allowed}
	csm.lock.Unlock()

	ignored := &ignoredProviders{providers: map[string]struct{}{}, currentEpoch: csm.atomicReadCurrentEpoch()}
	m, err := csm.tryGetConsumerSessionWithProviderFromBlockedProviderList(ctx, 1, ignored, cuForFirstRequest, servicedBlockNumber, "", nil, common.NO_STATE, 0, NewUsedProviders(nil))
	require.NoError(t, err)
	_, foundExcluded := m[excluded]
	require.False(t, foundExcluded, "non-whitelisted provider recovered from blocked list")
	_, foundAllowed := m[allowed]
	require.True(t, foundAllowed, "whitelisted blocked provider should still be recoverable")
}

// TestProviderWhitelist_GetSessionsRestrictsToWhitelisted drives the real GetSessions entry point
// (not an internal selection method) and asserts every returned session goes to the single
// whitelisted provider.
func TestProviderWhitelist_GetSessionsRestrictsToWhitelisted(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	csm.rpcEndpoint.Geolocation = geoEU
	require.NoError(t, csm.UpdateAllProviders(firstEpochHeight, createPairingList("", true), nil))
	allowed := providerStr + "5"
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(fmt.Sprintf(`{"chains":{%q:{"EU":[%q]}}}`, csm.rpcEndpoint.ChainID, allowed))))
	csm.SetProviderWhitelist(pw)

	for i := 0; i < 10; i++ {
		css, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "")
		require.NoError(t, err)
		require.Len(t, css, 1)
		for providerAddr, cs := range css {
			require.Equal(t, allowed, providerAddr)
			// release the session so the loop doesn't exhaust the per-provider session pool
			require.NoError(t, csm.OnSessionDone(cs.Session, servicedBlockNumber, cuForFirstRequest, time.Millisecond, cs.Session.CalculateExpectedLatency(2*time.Millisecond), 1, numberOfProviders, numberOfProviders, false, nil))
		}
	}
}

// TestProviderWhitelist_GetSessionsEmptyIntersectionNoResetStorm asserts that when the whitelist
// excludes every paired provider for this chain, GetSessions fails cleanly and, crucially, does
// NOT trigger validAddresses resets. This is the reason the filter is applied at the selection
// call site rather than inside CalculateAddonValidAddresses (which validatePairingListNotEmpty
// drives, and which would reset-loop on an empty set).
func TestProviderWhitelist_GetSessionsEmptyIntersectionNoResetStorm(t *testing.T) {
	ctx := context.Background()
	csm := CreateConsumerSessionManager()
	csm.rpcEndpoint.Geolocation = geoEU
	require.NoError(t, csm.UpdateAllProviders(firstEpochHeight, createPairingList("", true), nil))
	// Whitelist names only a provider on a different chain -> empty intersection for this chain.
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{"OTHERCHAIN":{"EU":["someoneElse"]}}}`)))
	csm.SetProviderWhitelist(pw)

	resetsBefore := csm.atomicReadNumberOfResets()
	for i := 0; i < 5; i++ {
		_, err := csm.GetSessions(ctx, 1, cuForFirstRequest, NewUsedProviders(nil), servicedBlockNumber, "", nil, common.NO_STATE, 0, "", "")
		require.Error(t, err) // no whitelisted providers for this chain -> clean failure, not a hang
	}
	require.Equal(t, resetsBefore, csm.atomicReadNumberOfResets(), "emptying the candidate set via whitelist must not trigger validAddresses resets")
}
