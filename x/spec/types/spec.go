package types

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Coin is a simple token denomination and amount pair, replacing sdk.Coin with
// no Cosmos SDK dependency. Custom UnmarshalJSON handles the protobuf JSON
// convention where Amount is encoded as a quoted string (e.g. "5000000000").
type Coin struct {
	Denom  string `json:"denom"`
	Amount int64  `json:"amount"`
}

// UnmarshalJSON handles both quoted-string and numeric Amount values.
// Protobuf JSON encoding uses strings for large integers: {"denom":"ulava","amount":"5000000000"}
func (c *Coin) UnmarshalJSON(b []byte) error {
	// Try a flexible intermediate representation first.
	var raw struct {
		Denom  string          `json:"denom"`
		Amount json.RawMessage `json:"amount"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	c.Denom = raw.Denom

	if len(raw.Amount) == 0 {
		c.Amount = 0
		return nil
	}

	// Try numeric first, then quoted string.
	if err := json.Unmarshal(raw.Amount, &c.Amount); err != nil {
		var s string
		if err2 := json.Unmarshal(raw.Amount, &s); err2 != nil {
			return fmt.Errorf("Coin.Amount: expected number or string, got %s", string(raw.Amount))
		}
		_, err3 := fmt.Sscanf(s, "%d", &c.Amount)
		if err3 != nil {
			return fmt.Errorf("Coin.Amount: cannot parse %q as int64: %w", s, err3)
		}
	}
	return nil
}

// FlexFloat64 is a float64 that accepts both string and numeric JSON values.
// Protobuf/cosmos JSON encoding uses strings for decimal types (e.g. "0.015").
type FlexFloat64 float64

func (f *FlexFloat64) UnmarshalJSON(b []byte) error {
	var num float64
	if err := json.Unmarshal(b, &num); err == nil {
		*f = FlexFloat64(num)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("FlexFloat64: expected number or string, got %s", string(b))
	}
	_, err := fmt.Sscanf(s, "%g", &num)
	if err != nil {
		return fmt.Errorf("FlexFloat64: cannot parse %q: %w", s, err)
	}
	*f = FlexFloat64(num)
	return nil
}

func (f FlexFloat64) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(f))
}

func (f FlexFloat64) Float64() float64 {
	return float64(f)
}

// IsValid reports whether the coin has a non-empty denomination.
func (c Coin) IsValid() bool { return c.Denom != "" }

// IsPositive reports whether the coin amount is strictly positive.
func (c Coin) IsPositive() bool { return c.Amount > 0 }

// Spec describes a single blockchain specification including all API
// collections, performance parameters, and governance metadata.
type Spec struct {
	Index                         string              `json:"index"                            mapstructure:"index"`
	Name                          string              `json:"name"                             mapstructure:"name"`
	Enabled                       bool                `json:"enabled,omitempty"`
	ReliabilityThreshold          uint32              `json:"reliability_threshold,omitempty"`
	DataReliabilityEnabled        bool                `json:"data_reliability_enabled,omitempty"`
	BlockDistanceForFinalizedData uint32              `json:"block_distance_for_finalized_data,omitempty"`
	BlocksInFinalizationProof     uint32              `json:"blocks_in_finalization_proof,omitempty"`
	AverageBlockTime              int64               `json:"average_block_time,omitempty"`
	AllowedBlockLagForQosSync     int64               `json:"allowed_block_lag_for_qos_sync,omitempty"`
	BlockLastUpdated              uint64              `json:"block_last_updated,omitempty"`
	MinStakeProvider              Coin                `json:"min_stake_provider"`
	ProvidersTypes                Spec_ProvidersTypes `json:"providers_types,omitempty"`
	Imports                       []string            `json:"imports,omitempty"`
	ApiCollections                []*ApiCollection    `json:"api_collections,omitempty"`
	Contributor                   []string            `json:"contributor,omitempty"`
	ContributorPercentage         *FlexFloat64        `json:"contributor_percentage,omitempty"`
	Shares                        uint64              `json:"shares,omitempty"`
	Identity                      string              `json:"identity,omitempty"`
}

// ---------------------------------------------------------------------------
// Spec getters (nil-safe value accessors matching the original protobuf API)
// ---------------------------------------------------------------------------

func (m *Spec) GetIndex() string {
	if m != nil {
		return m.Index
	}
	return ""
}

func (m *Spec) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Spec) GetEnabled() bool {
	if m != nil {
		return m.Enabled
	}
	return false
}

func (m *Spec) GetReliabilityThreshold() uint32 {
	if m != nil {
		return m.ReliabilityThreshold
	}
	return 0
}

func (m *Spec) GetDataReliabilityEnabled() bool {
	if m != nil {
		return m.DataReliabilityEnabled
	}
	return false
}

func (m *Spec) GetBlockDistanceForFinalizedData() uint32 {
	if m != nil {
		return m.BlockDistanceForFinalizedData
	}
	return 0
}

func (m *Spec) GetBlocksInFinalizationProof() uint32 {
	if m != nil {
		return m.BlocksInFinalizationProof
	}
	return 0
}

func (m *Spec) GetAverageBlockTime() int64 {
	if m != nil {
		return m.AverageBlockTime
	}
	return 0
}

func (m *Spec) GetAllowedBlockLagForQosSync() int64 {
	if m != nil {
		return m.AllowedBlockLagForQosSync
	}
	return 0
}

func (m *Spec) GetBlockLastUpdated() uint64 {
	if m != nil {
		return m.BlockLastUpdated
	}
	return 0
}

func (m *Spec) GetMinStakeProvider() Coin {
	if m != nil {
		return m.MinStakeProvider
	}
	return Coin{}
}

func (m *Spec) GetProvidersTypes() Spec_ProvidersTypes {
	if m != nil {
		return m.ProvidersTypes
	}
	return Spec_dynamic
}

func (m *Spec) GetImports() []string {
	if m != nil {
		return m.Imports
	}
	return nil
}

func (m *Spec) GetApiCollections() []*ApiCollection {
	if m != nil {
		return m.ApiCollections
	}
	return nil
}

func (m *Spec) GetContributor() []string {
	if m != nil {
		return m.Contributor
	}
	return nil
}

func (m *Spec) GetContributorPercentage() *FlexFloat64 {
	if m != nil {
		return m.ContributorPercentage
	}
	return nil
}

func (m *Spec) GetShares() uint64 {
	if m != nil {
		return m.Shares
	}
	return 0
}

func (m *Spec) GetIdentity() string {
	if m != nil {
		return m.Identity
	}
	return ""
}

// ---------------------------------------------------------------------------
// Spec methods (ported from original x/spec/types/spec.go without Cosmos deps)
// ---------------------------------------------------------------------------

// CombineCollections appends parent api collections that are not already
// present in the current spec, combining multiple parent entries for the same
// CollectionData into a single merged collection.
func (spec *Spec) CombineCollections(parentsCollections map[CollectionData][]*ApiCollection) error {
	if spec == nil {
		return fmt.Errorf("CombineCollections: spec is nil")
	}

	// Build a deterministically sorted list of keys.
	collectionDataList := make([]CollectionData, 0, len(parentsCollections))
	for key := range parentsCollections {
		collectionDataList = append(collectionDataList, key)
	}
	sort.SliceStable(collectionDataList, func(i, j int) bool {
		return collectionDataList[i].String() < collectionDataList[j].String()
	})

	for _, collectionData := range collectionDataList {
		collectionsToCombine := parentsCollections[collectionData]
		if len(collectionsToCombine) == 0 {
			return fmt.Errorf("collection with length 0 %v", collectionData)
		}

		var combined *ApiCollection
		var others []*ApiCollection
		for i := 0; i < len(collectionsToCombine); i++ {
			combined = collectionsToCombine[i]
			others = append(collectionsToCombine[:i:i], collectionsToCombine[i+1:]...)
			if combined.Enabled {
				break
			}
		}
		if combined == nil || !combined.Enabled {
			// No enabled collection to combine; skip.
			continue
		}

		if err := combined.CombineWithOthers(others, false, false); err != nil {
			return err
		}
		spec.ApiCollections = append(spec.ApiCollections, combined)
	}
	return nil
}

// ServicesMap returns the set of add-on and extension names declared across
// all api collections in the spec.
func (spec *Spec) ServicesMap() (addons, extensions map[string]struct{}) {
	addons = map[string]struct{}{}
	extensions = map[string]struct{}{}
	for _, apiCollection := range spec.ApiCollections {
		addons[apiCollection.CollectionData.AddOn] = struct{}{}
		for _, extension := range apiCollection.Extensions {
			extensions[extension.Name] = struct{}{}
		}
	}
	return
}
