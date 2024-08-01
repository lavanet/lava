package types

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/utils/decoder"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

func TestDecodeJsonPolicy(t *testing.T) {
	expectedPolicy := Policy{
		ChainPolicies: []ChainPolicy{
			{ChainId: "LAV1", Apis: []string{}},
			{ChainId: "ETH1", Apis: []string{"eth_blockNumber", "eth_accounts"}},
		},
		GeolocationProfile: 4,
		TotalCuLimit:       1000000,
		EpochCuLimit:       100000,
		MaxProvidersToPair: 3,
	}

	input := `
  {
    "policy": {
      "chain_policies": [
          {
              "chain_id": "LAV1",
              "apis": [
              ]
          },
          {
              "chain_id": "ETH1",
              "apis": [
                  "eth_blockNumber",
                  "eth_accounts"
              ]
          }
      ],
      "geolocation_profile": 4,
      "total_cu_limit": 1000000,
      "epoch_cu_limit": 100000,
      "max_providers_to_pair": 3,
  }
}`
	var policy Policy
	err := decoder.Decode(input, "policy", &policy, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, policy.Equal(expectedPolicy))
}

func TestDecodeJsonPlan(t *testing.T) {
	expectedPlan := Plan{
		Index:                    "to_delete_plan",
		Description:              "This plan has no restrictions",
		Type:                     "rpc",
		Price:                    types.NewCoin(commontypes.TokenDenom, types.NewIntFromUint64(100000)),
		AnnualDiscountPercentage: 20,
		AllowOveruse:             true,
		OveruseRate:              2,
		ProjectsLimit:            10,
		PlanPolicy: Policy{
			ChainPolicies: []ChainPolicy{
				{ChainId: "LAV1", Apis: []string{}},
				{ChainId: "ETH1", Apis: []string{"eth_blockNumber", "eth_accounts"}},
			},
			GeolocationProfile: 1,
			TotalCuLimit:       1000000,
			EpochCuLimit:       100000,
			MaxProvidersToPair: 3,
		},
	}
	input := `
  {
    "plan": {
        "index": "to_delete_plan",
        "description": "This plan has no restrictions",
        "type": "rpc",
        "price": {
            "denom": "ulava",
            "amount": "100000"
        },
        "annual_discount_percentage": 20,
        "allow_overuse": true,
        "overuse_rate": 2,
		"projects_limit": 10,
        "plan_policy": {
            "chain_policies": [
                {
                    "chain_id": "LAV1",
                    "apis": [
                    ]
                },
                {
                    "chain_id": "ETH1",
                    "apis": [
                        "eth_blockNumber",
                        "eth_accounts"
                    ]
                }
            ],
            "geolocation_profile": 1,
            "total_cu_limit": 1000000,
            "epoch_cu_limit": 100000,
            "max_providers_to_pair": 3
        }
    }
}`
	var plan Plan
	decoderHooks := []mapstructure.DecodeHookFunc{
		PriceDecodeHookFunc,
		PolicyEnumDecodeHookFunc,
	}

	err := decoder.Decode(input, "plan", &plan, decoderHooks, nil, nil)
	require.NoError(t, err)
	require.True(t, plan.Equal(expectedPlan))
}

func TestDecodePlanAddProposal(t *testing.T) {
	defaultGeo, ok := policyDefaultValues["geolocation_profile"].(int32)
	require.True(t, ok)
	defaultProvidersToPair, ok := policyDefaultValues["max_providers_to_pair"].(uint64)
	require.True(t, ok)

	expectedPlanPorposal := PlansAddProposal{
		Title:       "Add temporary to-delete plan proposal",
		Description: "A proposal of a temporary to-delete plan",
		Plans: []Plan{
			{
				Index:                    "to_delete_plan",
				Description:              "This plan has no restrictions",
				Type:                     "rpc",
				Price:                    types.NewCoin(commontypes.TokenDenom, types.NewIntFromUint64(100000)),
				AnnualDiscountPercentage: 20,
				AllowOveruse:             true,
				OveruseRate:              2,
				ProjectsLimit:            10,
				PlanPolicy: Policy{
					ChainPolicies: []ChainPolicy{
						{ChainId: "LAV1", Apis: []string{}},
						{ChainId: "ETH1", Apis: []string{"eth_blockNumber", "eth_accounts"}},
					},
					GeolocationProfile:    defaultGeo, // missing field
					TotalCuLimit:          1000000,
					EpochCuLimit:          100000,
					MaxProvidersToPair:    3,
					SelectedProvidersMode: SELECTED_PROVIDERS_MODE_MIXED,
					SelectedProviders:     []string{"lava@1wvn4slrf2r7cm92fnqdhvl3x470944uev92squ"},
				},
			},
			{
				Index:                    "to_delete_plan_2",
				Description:              "This plan has no restrictions",
				Type:                     "rpc",
				Price:                    types.NewCoin(commontypes.TokenDenom, types.NewIntFromUint64(100000)),
				AnnualDiscountPercentage: 20,
				AllowOveruse:             true,
				OveruseRate:              2,
				ProjectsLimit:            5,
				PlanPolicy: Policy{
					ChainPolicies: []ChainPolicy{
						{ChainId: "LAV1", Apis: []string{}},
						{ChainId: "ETH1", Apis: []string{"eth_blockNumber", "eth_accounts"}},
					},
					GeolocationProfile:    27,
					TotalCuLimit:          1000000,
					EpochCuLimit:          100000,
					MaxProvidersToPair:    defaultProvidersToPair, // missing field
					SelectedProvidersMode: 0,
					SelectedProviders:     []string{},
				},
			},
		},
	}

	expectedDeposit := "10000000ulava"
	input := `
	{
		"proposal": {
			"title": "Add temporary to-delete plan proposal",
			"description": "A proposal of a temporary to-delete plan",
			"plans": [
				{
					"index": "to_delete_plan",
					"dummy_plan_field": "def",
					"description": "This plan has no restrictions",
					"type": "rpc",
					"price": {
						"denom": "ulava",
						"amount": "100000"
					},
					"annual_discount_percentage": 20,
					"allow_overuse": true,
					"overuse_rate": 2,
					"projects_limit": 10,
					"plan_policy": {
						"chain_policies": [
							{
								"chain_id": "LAV1",
								"apis": [
								]
							},
							{
								"chain_id": "ETH1",
								"apis": [
									"eth_blockNumber",
									"eth_accounts"
								]
							}
						],
						"total_cu_limit": 1000000,
						"epoch_cu_limit": 100000,
						"max_providers_to_pair": 3,
						"selected_providers_mode": "MIXED",
						"selected_providers": [
							"lava@1wvn4slrf2r7cm92fnqdhvl3x470944uev92squ"
						],
						"dummy_policy_field": "abc",
						"dummy_policy_field_2": 34
					}
				},
				{
					"index": "to_delete_plan_2",
					"dummy_plan_field_2": "ghi",
					"description": "This plan has no restrictions",
					"type": "rpc",
					"price": {
						"denom": "ulava",
						"amount": "100000"
					},
					"annual_discount_percentage": 20,
					"allow_overuse": true,
					"overuse_rate": 2,
					"projects_limit": 5,
					"plan_policy": {
						"chain_policies": [
							{
								"chain_id": "LAV1",
								"apis": [
								]
							},
							{
								"chain_id": "ETH1",
								"apis": [
									"eth_blockNumber",
									"eth_accounts"
								]
							}
						],
						"geolocation_profile": 27,
						"epoch_cu_limit": 100000,
						"total_cu_limit": 1000000,
						"selected_providers_mode": 0,
						"selected_providers": [],
						"dummy_policy_field_3": 64
					}
				}
			]
		},
		"deposit": "10000000ulava"
	}`

	var (
		planPorposal PlansAddProposal
		deposit      string
		err          error
	)
	decoderHooks := []mapstructure.DecodeHookFunc{
		PriceDecodeHookFunc,
		PolicyEnumDecodeHookFunc,
	}

	var (
		unset  []string
		unused []string
	)

	err = decoder.Decode(input, "proposal", &planPorposal, decoderHooks, &unset, &unused)
	require.NoError(t, err)
	err = planPorposal.HandleUnsetPlanProposalFields(unset)
	require.NoError(t, err)
	require.True(t, planPorposal.Equal(expectedPlanPorposal))

	err = decoder.Decode(input, "deposit", &deposit, decoderHooks, nil, nil)
	require.NoError(t, err)
	require.True(t, deposit == expectedDeposit)
}

func TestPlanDelProposal(t *testing.T) {
	expectedProposal := PlansDelProposal{
		Title:       "Delete temporary (to-delete) plan proposal",
		Description: "A proposal to delete temporary (to-delete) plan",
		Plans:       []string{"to_delete_plan", "another_plan_delete"},
	}

	expectedDeposit := "10000000ulava"

	input := `
	{
		"proposal": {
			"title": "Delete temporary (to-delete) plan proposal",
			"description": "A proposal to delete temporary (to-delete) plan",
			"plans": [
				"to_delete_plan",
				"another_plan_delete"
			]
		},
		"deposit": "10000000ulava"
	}`

	var (
		planPorposal PlansDelProposal
		deposit      string
		err          error
	)

	err = decoder.Decode(input, "proposal", &planPorposal, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, planPorposal.Equal(expectedProposal))

	err = decoder.Decode(input, "deposit", &deposit, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, deposit == expectedDeposit)
}
