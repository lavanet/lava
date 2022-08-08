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
		ClientPaymentStorageList:               []ClientPaymentStorage{},
		EpochPaymentsList:                      []EpochPayments{},
		FixatedServicersToPairList:             []FixatedServicersToPair{},
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
	// Check for duplicated index in clientPaymentStorage
	clientPaymentStorageIndexMap := make(map[string]struct{})

	for _, elem := range gs.ClientPaymentStorageList {
		index := string(ClientPaymentStorageKey(elem.Index))
		if _, ok := clientPaymentStorageIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for clientPaymentStorage")
		}
		clientPaymentStorageIndexMap[index] = struct{}{}
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
	// Check for duplicated index in fixatedServicersToPair
	fixatedServicersToPairIndexMap := make(map[string]struct{})

	for _, elem := range gs.FixatedServicersToPairList {
		index := string(FixatedServicersToPairKey(elem.Index))
		if _, ok := fixatedServicersToPairIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for fixatedServicersToPair")
		}
		fixatedServicersToPairIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
