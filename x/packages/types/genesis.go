package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		PackageEntryList:       []PackageEntry{},
		PackageUniqueIndexList: []PackageUniqueIndex{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in packageEntry
	packageEntryIndexMap := make(map[string]struct{})

	for _, elem := range gs.PackageEntryList {
		index := string(PackageEntryKey(elem.PackageIndex))
		if _, ok := packageEntryIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for packageEntry")
		}
		packageEntryIndexMap[index] = struct{}{}
	}
	// Check for duplicated ID in packageUniqueIndex
	packageUniqueIndexIdMap := make(map[uint64]bool)
	packageUniqueIndexCount := gs.GetPackageUniqueIndexCount()
	for _, elem := range gs.PackageUniqueIndexList {
		if _, ok := packageUniqueIndexIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for packageUniqueIndex")
		}
		if elem.Id >= packageUniqueIndexCount {
			return fmt.Errorf("packageUniqueIndex id should be lower or equal than the last id")
		}
		packageUniqueIndexIdMap[elem.Id] = true
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
