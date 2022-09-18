package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		UniquePaymentStorageClientProviderList: []UniquePaymentStorageClientProvider{},
		ProviderPaymentStorageList:             []ProviderPaymentStorage{},
		EpochPaymentsList:                      []EpochPayments{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in uniquePaymentStorageClientProvider
	uniquePaymentStorageClientProviderIndexMap := make(map[string]struct{})

	for _, elem := range gs.UniquePaymentStorageClientProviderList {
		index := string(UniquePaymentStorageClientProviderKey(elem.Index))
		if _, ok := uniquePaymentStorageClientProviderIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for uniquePaymentStorageClientProvider")
		}
		uniquePaymentStorageClientProviderIndexMap[index] = struct{}{}
	}
	// Check for duplicated index in providerPaymentStorage
	providerPaymentStorageIndexMap := make(map[string]struct{})

	for _, elem := range gs.ProviderPaymentStorageList {
		index := string(ProviderPaymentStorageKey(elem.Index))
		if _, ok := providerPaymentStorageIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for providerPaymentStorage")
		}
		providerPaymentStorageIndexMap[index] = struct{}{}
	}
	// Check for duplicated index in epochPayments
	epochPaymentsIndexMap := make(map[string]struct{})

	for _, elem := range gs.EpochPaymentsList {
		index := string(EpochPaymentsKey(elem.Index))
		if _, ok := epochPaymentsIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for epochPayments")
		}
		epochPaymentsIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
