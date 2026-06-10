package lavasession

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/lavanet/lava/v5/utils"
	planstypes "github.com/lavanet/lava/v5/x/plans/types"
)

// errNotWhitelistDocument marks a file that is valid JSON but is not a whitelist (it omits the
// "chains" key). During a multi-file load these are skipped so a shared/mixed repo of unrelated
// configs doesn't break the load. It is deliberately distinct from a malformed-JSON error (or an
// invalid geolocation), which is treated as a hard failure: a corrupt file must not silently drop
// its chains from the snapshot.
var errNotWhitelistDocument = errors.New(`not a provider whitelist document (missing "chains" field)`)

// providerWhitelistJSON is the JSON schema of the consumer provider whitelist, as fetched from
// GitHub or a local file. It is keyed chain -> geolocation -> the providers allowed to serve that
// chain from that region:
//
//	{
//	  "chains": {
//	    "ETH1": { "EU": ["lava@1abc...", "lava@1def..."], "AS": ["lava@1ghi..."] },
//	    "LAV1": { "EU": ["lava@1abc..."] }
//	  }
//	}
//
// The geolocation keys are the on-chain region codes parsed by planstypes.ParseGeoEnum: a name
// ("EU", "AS", "USC"/"USE"/"USW", "AF", "AU"), a raw bitmask ("2"), a comma-combined set ("EU,AS"),
// or "GL" meaning "any region". Reusing that parser makes the list speak the same geolocation
// vocabulary as the consumer's own --geolocation flag and every on-chain policy.
//
// Chains is a pointer so we can distinguish a document that omits the "chains" key (not a whitelist
// file at all -> rejected) from one that sets it to an empty object (an intentional "allow nobody"
// whitelist -> accepted).
type providerWhitelistJSON struct {
	Chains *map[string]map[string][]string `json:"chains"`
	// Providers detects the OLD pre-geolocation schema (a top-level "providers" array). It is never
	// parsed; its only purpose is to fail an un-migrated file loudly instead of silently treating it
	// as "not a whitelist" and falling through to passthrough (relay to everyone).
	Providers *json.RawMessage `json:"providers"`
}

// whitelistData is the immutable, lookup-optimized index built from the parsed JSON:
// chainID -> geolocation code -> set of allowed provider addresses.
type whitelistData struct {
	byChainGeo map[string]map[int32]map[string]struct{}
}

// geoMatch reports whether a consumer in consumerGeo should be served by an entry whitelisted for
// regionCode. A "GL" entry matches every region; otherwise the consumer's geolocation must share a
// bit with the region (bitwise overlap), which reduces to plain equality for a single discrete
// region (the common per-region-gateway case).
func geoMatch(consumerGeo uint64, regionCode int32) bool {
	if regionCode == int32(planstypes.Geolocation_GL) {
		return true
	}
	return consumerGeo&uint64(regionCode) != 0
}

// isAllowed reports whether providerAddr may serve chainID for a consumer in consumerGeo. A chain
// has at most one bucket per geolocation, so this per-relay scan is over a handful of regions and
// is effectively constant time.
func (wd *whitelistData) isAllowed(chainID string, consumerGeo uint64, providerAddr string) bool {
	geoSets, ok := wd.byChainGeo[chainID]
	if !ok {
		return false
	}
	for regionCode, addrs := range geoSets {
		if !geoMatch(consumerGeo, regionCode) {
			continue
		}
		if _, ok := addrs[providerAddr]; ok {
			return true
		}
	}
	return false
}

// entryCount returns the number of distinct (chain, geolocation, provider) entries, used for logging.
func (wd *whitelistData) entryCount() int {
	count := 0
	for _, geoSets := range wd.byChainGeo {
		for _, addrs := range geoSets {
			count += len(addrs)
		}
	}
	return count
}

// ProviderWhitelist is a thread-safe, hourly-refreshed allowlist that restricts which providers the
// consumer is willing to relay to, per chain AND per geolocation. A single instance is shared across
// all per-chain ConsumerSessionManagers; each manager queries it with its own ChainID and the
// consumer's configured geolocation.
//
// Semantics:
//   - Until the first successful load it is "not loaded": IsAllowed returns true (passthrough), so
//     a startup fetch failure keeps the consumer relaying to everyone rather than going dark.
//   - Once loaded, IsAllowed returns true only for (chainID, geolocation, providerAddr) tuples
//     present in the list. A loaded-but-empty list therefore allows nobody (intentional whitelist
//     semantics). A later refresh failure does NOT clear the snapshot, so the last-known-good list
//     stays in place.
//
// The active snapshot is held in an atomic.Pointer so the read hot path (IsAllowed, called
// per-candidate-provider per-relay) takes no locks, and the hourly refresh swaps it atomically.
type ProviderWhitelist struct {
	data atomic.Pointer[whitelistData]
}

// NewProviderWhitelist returns a whitelist in the "not loaded" (passthrough) state. Until the first
// successful load the consumer relays to everyone; once loaded a refresh failure keeps the
// last-known-good snapshot.
func NewProviderWhitelist() *ProviderWhitelist {
	return &ProviderWhitelist{}
}

// Enabled reports whether a whitelist has been successfully loaded at least once.
func (pw *ProviderWhitelist) Enabled() bool {
	return pw.data.Load() != nil
}

// snapshot returns the current whitelist index, or nil if no whitelist has loaded yet. Callers
// filtering a list should snapshot once and reuse it across elements rather than calling
// IsAllowed per element.
func (pw *ProviderWhitelist) snapshot() *whitelistData {
	return pw.data.Load()
}

// IsAllowed reports whether the consumer (in consumerGeo) may relay to providerAddr for chainID. It
// returns true (passthrough) when no whitelist has been loaded yet, so a startup fetch failure keeps
// the consumer relaying to everyone rather than going dark.
func (pw *ProviderWhitelist) IsAllowed(chainID string, consumerGeo uint64, providerAddr string) bool {
	data := pw.data.Load()
	if data == nil {
		return true // not loaded yet -> passthrough (allow all until the first successful load)
	}
	return data.isAllowed(chainID, consumerGeo, providerAddr)
}

// UpdateFromBytes parses a single whitelist JSON document and atomically replaces the active
// snapshot. On error the previous snapshot is left intact.
func (pw *ProviderWhitelist) UpdateFromBytes(content []byte) error {
	index, err := parseWhitelist(content)
	if err != nil {
		return err
	}
	pw.store(index)
	return nil
}

// UpdateFromFiles parses each provided file as a whitelist document and atomically replaces the
// active snapshot with the union of their entries. Files that are valid JSON but are not whitelist
// documents (no "chains" key) are skipped with a warning, so a shared/mixed repo does not break the
// load. Any other parse failure (malformed JSON or an invalid geolocation) is treated as a hard
// error: the previous snapshot is left intact and an error is returned, so a corrupt file can't
// silently drop its chains from the snapshot. If no file yields a valid whitelist, the previous
// snapshot is likewise left intact and an error is returned.
func (pw *ProviderWhitelist) UpdateFromFiles(files map[string][]byte) error {
	merged := map[string]map[int32]map[string]struct{}{}
	parsedAny := false
	for fileURL, content := range files {
		index, err := parseWhitelist(content)
		if err != nil {
			if errors.Is(err, errNotWhitelistDocument) {
				utils.LavaFormatWarning("skipping non-whitelist file in provider whitelist source", err, utils.LogAttr("file", fileURL))
				continue
			}
			// Malformed/invalid whitelist file: fail the whole refresh and keep the last-known-good
			// snapshot rather than swapping in a partial union that silently omits this file's chains.
			return fmt.Errorf("malformed provider whitelist file %q: %w", fileURL, err)
		}
		parsedAny = true
		for chainID, geoSets := range index.byChainGeo {
			if merged[chainID] == nil {
				merged[chainID] = map[int32]map[string]struct{}{}
			}
			for geo, addrs := range geoSets {
				if merged[chainID][geo] == nil {
					merged[chainID][geo] = map[string]struct{}{}
				}
				for addr := range addrs {
					merged[chainID][geo][addr] = struct{}{}
				}
			}
		}
	}
	if !parsedAny {
		return fmt.Errorf("no valid provider whitelist files found (out of %d fetched)", len(files))
	}
	pw.store(&whitelistData{byChainGeo: merged})
	return nil
}

// store atomically swaps in a new snapshot and logs the result. An empty (but loaded) whitelist is
// logged loudly because it causes the consumer to relay to no providers.
func (pw *ProviderWhitelist) store(data *whitelistData) {
	pw.data.Store(data)
	entries := data.entryCount()
	if entries == 0 {
		utils.LavaFormatWarning("provider whitelist loaded but EMPTY - the consumer will relay to NO providers", nil)
		return
	}
	utils.LavaFormatInfo("provider whitelist loaded",
		utils.LogAttr("chains", len(data.byChainGeo)),
		utils.LogAttr("allowed_provider_chain_geo_entries", entries),
	)
}

// parseWhitelist parses a single whitelist JSON document into a lookup index. It returns an error
// for malformed JSON, for a document that omits the "chains" key (i.e. is not a whitelist), or for
// an unparseable geolocation code.
func parseWhitelist(content []byte) (*whitelistData, error) {
	var doc providerWhitelistJSON
	if err := json.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse provider whitelist JSON: %w", err)
	}
	if doc.Chains == nil {
		// An old-schema whitelist (top-level "providers", no "chains") is a stale file that must be
		// migrated, not an unrelated file to skip. Fail it loudly: errNotWhitelistDocument would be
		// skipped in a multi-file load, leaving the consumer in passthrough (relaying to everyone).
		if doc.Providers != nil {
			return nil, fmt.Errorf(`provider whitelist uses the old pre-geolocation schema (top-level "providers"); migrate it to the chain -> geolocation -> providers schema (see docs/provider-whitelist.md)`)
		}
		return nil, errNotWhitelistDocument
	}

	byChainGeo := map[string]map[int32]map[string]struct{}{}
	for chainID, regions := range *doc.Chains {
		if chainID == "" {
			continue
		}
		for region, addrs := range regions {
			if region == "" {
				continue
			}
			// A typo'd region (e.g. "Asia" instead of "AS", or "US" which is not a region code) is a
			// hard error rather than a silent skip: quietly dropping it would stop relaying for that
			// region, which is exactly the misconfiguration this whitelist exists to prevent.
			geoCode, err := planstypes.ParseGeoEnum(region)
			if err != nil {
				return nil, fmt.Errorf("invalid geolocation %q for chain %q in provider whitelist: %w", region, chainID, err)
			}
			for _, addr := range addrs {
				if addr == "" {
					continue
				}
				if byChainGeo[chainID] == nil {
					byChainGeo[chainID] = map[int32]map[string]struct{}{}
				}
				if byChainGeo[chainID][geoCode] == nil {
					byChainGeo[chainID][geoCode] = map[string]struct{}{}
				}
				byChainGeo[chainID][geoCode][addr] = struct{}{}
			}
		}
	}
	return &whitelistData{byChainGeo: byChainGeo}, nil
}
