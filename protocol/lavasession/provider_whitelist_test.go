package lavasession

import (
	"testing"

	planstypes "github.com/lavanet/lava/v5/x/plans/types"
	"github.com/stretchr/testify/require"
)

// region codes used by the tests, matching the on-chain geolocation enum.
const (
	geoEU = uint64(planstypes.Geolocation_EU) // 2
	geoAS = uint64(planstypes.Geolocation_AS) // 32
)

const sampleWhitelistJSON = `{
  "chains": {
    "ETH1": { "EU": ["provider0", "provider1"] },
    "LAV1": { "EU": ["provider0"] },
    "NEAR": { "AS": ["provider2"] }
  }
}`

func TestProviderWhitelist_NotLoadedIsPassthrough(t *testing.T) {
	pw := NewProviderWhitelist()
	require.False(t, pw.Enabled())
	// Until a list loads, everything is allowed (current relay behavior).
	require.True(t, pw.IsAllowed("ETH1", geoEU, "anyProvider"))
	require.True(t, pw.IsAllowed("anyChain", geoAS, "anyProvider"))
}

func TestProviderWhitelist_IsAllowedHitAndMiss(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(sampleWhitelistJSON)))
	require.True(t, pw.Enabled())

	// Hits: (provider, chain, region) tuples present in the list.
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.True(t, pw.IsAllowed("LAV1", geoEU, "provider0"))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider1"))
	require.True(t, pw.IsAllowed("NEAR", geoAS, "provider2"))

	// Misses: provider present but not on that chain.
	require.False(t, pw.IsAllowed("LAV1", geoEU, "provider1")) // provider1 is only on ETH1
	require.False(t, pw.IsAllowed("NEAR", geoAS, "provider0"))
	// Misses: right chain and provider, but wrong region.
	require.False(t, pw.IsAllowed("ETH1", geoAS, "provider0")) // provider0 is EU-only for ETH1
	require.False(t, pw.IsAllowed("NEAR", geoEU, "provider2")) // provider2 is AS-only for NEAR
	// Miss: provider not in the list at all.
	require.False(t, pw.IsAllowed("ETH1", geoEU, "providerX"))
	// Miss: chain not in the list at all.
	require.False(t, pw.IsAllowed("BTC", geoEU, "provider0"))
}

func TestProviderWhitelist_PerChainIsolation(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(sampleWhitelistJSON)))
	// The same provider is allowed on one chain and denied on another.
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.False(t, pw.IsAllowed("NEAR", geoEU, "provider0"))
}

func TestProviderWhitelist_PerGeolocationIsolation(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(sampleWhitelistJSON)))
	// The same (provider, chain) is allowed in its region and denied from another region. This is
	// the core MAG-1987 requirement: an EU gateway and an AS gateway gate the same chain differently.
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.False(t, pw.IsAllowed("ETH1", geoAS, "provider0"))
}

func TestProviderWhitelist_SameProviderMultipleRegions(t *testing.T) {
	pw := NewProviderWhitelist()
	// Pau's "Provider1 in both EU and Asia" case: one provider listed under two regions for a chain.
	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{"ETH1":{"EU":["provider0","provider1"],"AS":["provider1"]}}}`)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider1"))
	require.True(t, pw.IsAllowed("ETH1", geoAS, "provider1"))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.False(t, pw.IsAllowed("ETH1", geoAS, "provider0")) // provider0 is EU-only
}

func TestProviderWhitelist_GLRegionMatchesAnyGeo(t *testing.T) {
	pw := NewProviderWhitelist()
	// "GL" means "allowed from any region" — an escape hatch that recovers chain-only behavior.
	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{"ETH1":{"GL":["provider0"]}}}`)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.True(t, pw.IsAllowed("ETH1", geoAS, "provider0"))
	require.True(t, pw.IsAllowed("ETH1", 0, "provider0")) // even an unset consumer geo matches GL
	require.False(t, pw.IsAllowed("ETH1", geoEU, "provider1"))
}

func TestProviderWhitelist_NamedAndNumericRegionsEquivalent(t *testing.T) {
	pw := NewProviderWhitelist()
	// A region key may be the raw bitmask ("2") instead of the name ("EU"); both parse identically.
	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{"ETH1":{"2":["provider0"]}}}`)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.False(t, pw.IsAllowed("ETH1", geoAS, "provider0"))
}

func TestProviderWhitelist_CombinedRegionKeyMatchesEither(t *testing.T) {
	pw := NewProviderWhitelist()
	// A comma-combined region key whitelists the provider for any of the listed regions.
	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{"ETH1":{"EU,AS":["provider0"]}}}`)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.True(t, pw.IsAllowed("ETH1", geoAS, "provider0"))
	require.False(t, pw.IsAllowed("ETH1", uint64(planstypes.Geolocation_AF), "provider0"))
}

func TestProviderWhitelist_InvalidRegionRejectedKeepsPrevious(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(sampleWhitelistJSON)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))

	// "US" is not a region code (the codes are USC/USE/USW) — a hard error, last-known-good kept.
	require.Error(t, pw.UpdateFromBytes([]byte(`{"chains":{"ETH1":{"US":["provider0"]}}}`)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
}

func TestProviderWhitelist_EmptyListAllowsNobody(t *testing.T) {
	pw := NewProviderWhitelist()
	// A loaded-but-empty whitelist is "allow nobody" (intentional whitelist semantics).
	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{}}`)))
	require.True(t, pw.Enabled())
	require.False(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
}

func TestProviderWhitelist_MalformedKeepsPrevious(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(sampleWhitelistJSON)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))

	// A malformed update must error and leave the last-known-good snapshot intact.
	require.Error(t, pw.UpdateFromBytes([]byte(`{ not valid json `)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
}

func TestProviderWhitelist_MissingChainsFieldRejected(t *testing.T) {
	pw := NewProviderWhitelist()
	// A document without a "chains" key is not a whitelist (e.g. a spec file) and is rejected, rather
	// than being treated as an empty (allow-nobody) whitelist.
	require.Error(t, pw.UpdateFromBytes([]byte(`{"proposal":{"specs":[]}}`)))
	require.False(t, pw.Enabled())
}

func TestProviderWhitelist_OldSchemaRejectedLoudly(t *testing.T) {
	oldSchema := []byte(`{"providers":[{"address":"provider0","chains":["ETH1"]}]}`)

	// A single-source load of an un-migrated file errors (so the fetcher logs it), rather than
	// silently loading nothing.
	pw := NewProviderWhitelist()
	require.Error(t, pw.UpdateFromBytes(oldSchema))
	require.False(t, pw.Enabled())

	// In a multi-file load it must fail the WHOLE refresh (it is NOT skipped like an unrelated file),
	// so an old file can't quietly drop the consumer into passthrough.
	require.Error(t, pw.UpdateFromFiles(map[string][]byte{"old.json": oldSchema}))
}

func TestProviderWhitelist_AtomicSwapReplacesList(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{"ETH1":{"EU":["provider0"]}}}`)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.False(t, pw.IsAllowed("ETH1", geoEU, "provider1"))

	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{"ETH1":{"EU":["provider1"]}}}`)))
	require.False(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider1"))
}

func TestProviderWhitelist_UpdateFromFilesUnions(t *testing.T) {
	pw := NewProviderWhitelist()
	files := map[string][]byte{
		"a.json": []byte(`{"chains":{"ETH1":{"EU":["provider0"]}}}`),
		"b.json": []byte(`{"chains":{"ETH1":{"AS":["provider1"]},"LAV1":{"EU":["provider1"]}}}`),
	}
	require.NoError(t, pw.UpdateFromFiles(files))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.True(t, pw.IsAllowed("ETH1", geoAS, "provider1"))
	require.True(t, pw.IsAllowed("LAV1", geoEU, "provider1"))
	// The union keeps the regions distinct: provider1 was only listed for ETH1 in AS.
	require.False(t, pw.IsAllowed("ETH1", geoEU, "provider1"))
	require.False(t, pw.IsAllowed("LAV1", geoEU, "provider0"))
}

func TestProviderWhitelist_UpdateFromFilesSkipsNonWhitelist(t *testing.T) {
	pw := NewProviderWhitelist()
	files := map[string][]byte{
		"whitelist.json": []byte(`{"chains":{"ETH1":{"EU":["provider0"]}}}`),
		"spec.json":      []byte(`{"proposal":{"specs":[{"index":"ETH1"}]}}`), // valid JSON, no "chains" -> skipped
	}
	// Succeeds because at least one file is a valid whitelist; files that are valid JSON but aren't
	// whitelist documents are skipped (mixed-repo support).
	require.NoError(t, pw.UpdateFromFiles(files))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
}

// TestProviderWhitelist_UpdateFromFilesMalformedKeepsPrevious pins the all-or-nothing contract for
// a corrupt file: a file that fetched OK but is malformed JSON fails the whole refresh and leaves
// the last-known-good snapshot intact, instead of swapping in a partial union that silently drops
// the good file's chains. (A malformed JSON file is distinct from a valid-JSON non-whitelist file,
// which is still skipped — see TestProviderWhitelist_UpdateFromFilesSkipsNonWhitelist.)
func TestProviderWhitelist_UpdateFromFilesMalformedKeepsPrevious(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(sampleWhitelistJSON)))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))

	files := map[string][]byte{
		"good.json":    []byte(`{"chains":{"NEAR":{"AS":["providerX"]}}}`),
		"corrupt.json": []byte(`not json at all`), // malformed -> hard failure, no swap
	}
	require.Error(t, pw.UpdateFromFiles(files))
	// The previous snapshot is untouched: the good file's NEAR provider was NOT swapped in, and the
	// original ETH1 provider is still allowed.
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.False(t, pw.IsAllowed("NEAR", geoAS, "providerX"))
}

func TestProviderWhitelist_UpdateFromFilesAllInvalidKeepsPrevious(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(sampleWhitelistJSON)))

	files := map[string][]byte{
		"spec.json":    []byte(`{"proposal":{"specs":[]}}`),
		"garbage.json": []byte(`nope`),
	}
	// No file is a valid whitelist -> error, and the previous snapshot is preserved.
	require.Error(t, pw.UpdateFromFiles(files))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
}

func TestProviderWhitelist_IgnoresEmptyAddressChainAndRegion(t *testing.T) {
	pw := NewProviderWhitelist()
	require.NoError(t, pw.UpdateFromBytes([]byte(`{"chains":{
		"":{"EU":["provider0"]},
		"ETH1":{"":["provider0"],"EU":["","provider1"]}
	}}`)))
	// Empty chain id, empty region key, and empty address all contribute nothing.
	require.True(t, pw.Enabled())
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider1"))
	require.False(t, pw.IsAllowed("ETH1", geoEU, "provider0")) // only appeared under empty chain/region
	require.False(t, pw.IsAllowed("ETH1", geoEU, ""))
}
