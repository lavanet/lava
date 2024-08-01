package types

import (
	"fmt"
	"strings"

	fixationtypes "github.com/lavanet/lava/x/fixationstore/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		UniqueEpochSessions:      []UniqueEpochSessionGenesis{},
		ProviderEpochCus:         []ProviderEpochCuGenesis{},
		ProviderConsumerEpochCus: []ProviderConsumerEpochCuGenesis{},
		BadgeUsedCuList:          []BadgeUsedCu{},
		Reputations:              []ReputationGenesis{},
		BadgesTS:                 *timerstoretypes.DefaultGenesis(),
		ReputationScores:         *fixationtypes.DefaultGenesis(),
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in UniqueEpochSession
	UniqueEpochSessionsProviderIndexMap := make(map[string]struct{})

	for _, elem := range gs.UniqueEpochSessions {
		index := string(UniqueEpochSessionKey(elem.Epoch, elem.Provider, elem.ChainId, elem.Project, elem.SessionId))
		if _, ok := UniqueEpochSessionsProviderIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for UniqueEpochSession")
		}
		UniqueEpochSessionsProviderIndexMap[index] = struct{}{}
	}
	// Check for duplicated index in ProviderEpochCu
	providerEpochCusIndexMap := make(map[string]struct{})
	for _, elem := range gs.ProviderEpochCus {
		index := string(ProviderEpochCuKey(elem.Epoch, elem.Provider, elem.ChainId))
		if _, ok := providerEpochCusIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for ProviderEpochCu")
		}
		providerEpochCusIndexMap[index] = struct{}{}
	}

	// Check for duplicated index in ProviderEpochCu
	providerEpochComplainerCusIndexMap := make(map[string]struct{})
	for _, elem := range gs.ProviderEpochComplainedCus {
		index := string(ProviderEpochCuKey(elem.Epoch, elem.Provider, elem.ChainId))
		if _, ok := providerEpochComplainerCusIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for ProviderEpochCu")
		}
		providerEpochComplainerCusIndexMap[index] = struct{}{}
	}

	// Check for duplicated index in ProviderConsumerEpochCu
	providerConsumerEpochCuIndexMap := make(map[string]struct{})

	for _, elem := range gs.ProviderConsumerEpochCus {
		index := string(ProviderConsumerEpochCuKey(elem.Epoch, elem.Provider, elem.Project, elem.ChainId))
		if _, ok := providerConsumerEpochCuIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for ProviderConsumerEpochCu")
		}
		providerConsumerEpochCuIndexMap[index] = struct{}{}
	}

	// check the badgeUsedCuIndex map is empty
	if len(gs.BadgeUsedCuList) > 0 {
		return fmt.Errorf("badgeUsedCuList is not empty")
	}

	reputationsIndexMap := map[string]struct{}{}
	for _, elem := range gs.Reputations {
		index := strings.Join([]string{elem.ChainId, elem.Cluster, elem.Provider}, " ")
		if _, ok := reputationsIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for Reputations")
		}
		reputationsIndexMap[index] = struct{}{}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
