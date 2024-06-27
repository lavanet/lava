package common

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	commonconsts "github.com/lavanet/lava/testutil/common/consts"
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
	spec.MinStakeProvider = sdk.NewCoin(commonconsts.TestTokenDenom, sdk.NewInt(1000))
	spec.ApiCollections = []*spectypes.ApiCollection{{Enabled: true, CollectionData: spectypes.CollectionData{ApiInterface: spectypes.APIInterfaceJsonRPC, Type: "GET"}, Apis: []*spectypes.Api{{Name: specName + "API", ComputeUnits: 100, Enabled: true}}}}
	spec.BlocksInFinalizationProof = 10
	spec.Shares = 1
	spec.AverageBlockTime = 10000
	spec.AllowedBlockLagForQosSync = 5
	return spec
}

func CreateMockPlan() plantypes.Plan {
	plan := plantypes.Plan{
		Index:                    "free",
		Description:              "plan for testing",
		Type:                     "rpc",
		Block:                    100,
		Price:                    sdk.NewCoin(commonconsts.TestTokenDenom, sdk.NewInt(100)),
		AllowOveruse:             true,
		OveruseRate:              10,
		AnnualDiscountPercentage: 20,
		PlanPolicy:               CreateMockPolicy(),
		ProjectsLimit:            10,
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
