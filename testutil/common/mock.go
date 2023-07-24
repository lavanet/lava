package common

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func CreateMockSpec() spectypes.Spec {
	specName := "mockspec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.ReliabilityThreshold = 4294967295
	spec.BlockDistanceForFinalizedData = 0
	spec.DataReliabilityEnabled = true
	spec.MinStakeClient = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(100))
	spec.MinStakeProvider = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1000))
	spec.ApiCollections = []*spectypes.ApiCollection{{Enabled: true, CollectionData: spectypes.CollectionData{ApiInterface: "stub", Type: "GET"}, Apis: []*spectypes.Api{{Name: specName + "API", ComputeUnits: 100, Enabled: true}}}}
	spec.BlockDistanceForFinalizedData = 0
	return spec
}

func CreateMockPlan() plantypes.Plan {
	plan := plantypes.Plan{
		Index:                    "mock_plan",
		Description:              "plan for testing",
		Type:                     "rpc",
		Block:                    100,
		Price:                    sdk.NewCoin("ulava", sdk.NewInt(100)),
		AllowOveruse:             true,
		OveruseRate:              10,
		AnnualDiscountPercentage: 20,
		PlanPolicy:               CreateMockPolicy(),
	}

	return plan
}

func CreateMockPolicy() plantypes.Policy {
	policy := plantypes.Policy{
		TotalCuLimit:       100000,
		EpochCuLimit:       10000,
		MaxProvidersToPair: 3,
		GeolocationProfile: 1,
	}

	return policy
}
