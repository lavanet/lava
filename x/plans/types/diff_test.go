package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v5/utils/common/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

func createDiffTestPlan() Plan {
	return Plan{
		Index:                    "mock",
		Block:                    100,
		Description:              "a plan for testing",
		Type:                     "rpc",
		Price:                    sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(100)),
		AllowOveruse:             false,
		OveruseRate:              0,
		AnnualDiscountPercentage: 10,
		ProjectsLimit:            5,
		AllowedBuyers:            []string{"alice", "bob"},
		PlanPolicy: Policy{
			ChainPolicies: []ChainPolicy{
				{ChainId: "ETH1", Apis: []string{"eth_blockNumber"}},
				{
					ChainId: "NEAR",
					Requirements: []ChainRequirement{
						{
							Collection: spectypes.CollectionData{ApiInterface: "jsonrpc", Type: "POST"},
							Extensions: []string{"archive"},
							Mixed:      true,
						},
					},
				},
			},
			GeolocationProfile:    int32(Geolocation_GL),
			TotalCuLimit:          1000,
			EpochCuLimit:          100,
			MaxProvidersToPair:    3,
			SelectedProvidersMode: SELECTED_PROVIDERS_MODE_ALLOWED,
			SelectedProviders:     []string{},
		},
	}
}

func TestDiffPlansNoChanges(t *testing.T) {
	plan := createDiffTestPlan()
	require.Empty(t, DiffPlans(plan, plan))

	// the block field holds the version's creation height, so it must not be
	// considered a content change
	newVersion := createDiffTestPlan()
	newVersion.Block = plan.Block + 50
	require.Empty(t, DiffPlans(plan, newVersion))
}

func TestDiffPlansFields(t *testing.T) {
	prev := createDiffTestPlan()

	next := createDiffTestPlan()
	next.Description = "an updated plan"
	next.Type = "web3"
	next.Price = sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(200))
	next.AllowOveruse = true
	next.OveruseRate = 2
	next.AnnualDiscountPercentage = 25
	next.ProjectsLimit = 10
	next.AllowedBuyers = []string{"bob", "carol"}
	next.PlanPolicy.GeolocationProfile = int32(Geolocation_EU) | int32(Geolocation_USE)
	next.PlanPolicy.TotalCuLimit = 2000
	next.PlanPolicy.EpochCuLimit = 200
	next.PlanPolicy.MaxProvidersToPair = 5
	next.PlanPolicy.SelectedProvidersMode = SELECTED_PROVIDERS_MODE_MIXED
	next.PlanPolicy.SelectedProviders = []string{"validProvider"}

	expected := []string{
		`description: "a plan for testing" -> "an updated plan"`,
		`type: "rpc" -> "web3"`,
		`price: 100ulava -> 200ulava`,
		`allow_overuse: false -> true`,
		`overuse_rate: 0 -> 2`,
		`annual_discount_percentage: 10 -> 25`,
		`projects_limit: 5 -> 10`,
		`allowed_buyers: added [carol], removed [alice]`,
		`plan_policy.geolocation_profile: GL -> EU,USE`,
		`plan_policy.total_cu_limit: 1000 -> 2000`,
		`plan_policy.epoch_cu_limit: 100 -> 200`,
		`plan_policy.max_providers_to_pair: 3 -> 5`,
		`plan_policy.selected_providers_mode: ALLOWED -> MIXED`,
		`plan_policy.selected_providers: added [validProvider]`,
	}
	require.Equal(t, expected, DiffPlans(prev, next))
}

func TestDiffPlansChainPolicies(t *testing.T) {
	prev := createDiffTestPlan()

	next := createDiffTestPlan()
	// change the requirements of an existing chain, remove a chain and add a chain
	next.PlanPolicy.ChainPolicies = []ChainPolicy{
		{
			ChainId: "NEAR",
			Requirements: []ChainRequirement{
				{
					Collection: spectypes.CollectionData{ApiInterface: "jsonrpc", Type: "POST", AddOn: "debug"},
					Mixed:      true,
				},
			},
		},
		{ChainId: WILDCARD_CHAIN_POLICY},
	}

	expected := []string{
		`plan_policy.chain_policies: changed NEAR{requirements: [jsonrpc/POST ext=[archive] mixed]} -> NEAR{requirements: [jsonrpc/POST addon=debug mixed]}`,
		`plan_policy.chain_policies: added *{}`,
		`plan_policy.chain_policies: removed ETH1{apis: [eth_blockNumber]}`,
	}
	require.Equal(t, expected, DiffPlans(prev, next))
}

func TestDiffPlansIndex(t *testing.T) {
	prev := createDiffTestPlan()
	next := createDiffTestPlan()
	next.Index = "mock2"

	require.Equal(t, []string{`index: "mock" -> "mock2"`}, DiffPlans(prev, next))
}
