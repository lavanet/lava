package rpcconsumer

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/stretchr/testify/require"
)

func TestRelayError(t *testing.T) {
	expectedValue := "Expected Error"
	testStruct := []struct {
		name        string
		relayErrors RelayErrors
	}{
		{
			name: "test stake majority error reply",
			relayErrors: RelayErrors{
				onFailureMergeAll: true,
				relayErrors: []RelayError{
					{
						err: fmt.Errorf("test1"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf("test2"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 20),
						},
					},
					{
						err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 30),
						},
					},
					{
						err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 40),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 50),
						},
					},
				},
			},
		},
		{
			name: "test qos majority error reply",
			relayErrors: RelayErrors{
				onFailureMergeAll: true,
				relayErrors: []RelayError{
					{
						err: fmt.Errorf("test1"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.MustNewDecFromStr("0.5"),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf("test1"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.MustNewDecFromStr("0.25"),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.MustNewDecFromStr("0.6"),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.MustNewDecFromStr("0.7"),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.MustNewDecFromStr("0.7"),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.MustNewDecFromStr("0.7"),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.MustNewDecFromStr("0.8"),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
				},
			},
		},
		{
			name: "test text majority over score majority",
			relayErrors: RelayErrors{
				onFailureMergeAll: true,
				relayErrors: []RelayError{
					{
						err: fmt.Errorf("test1"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 1000),
						},
					},
					{
						err: fmt.Errorf("test2"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 1000),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.ZeroDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 0),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.ZeroDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 0),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.ZeroDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 0),
						},
					},
				},
			},
		},
		{
			name: "test majority of error body",
			relayErrors: RelayErrors{
				onFailureMergeAll: true,
				relayErrors: []RelayError{
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 20),
						},
					},
					{
						err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 30),
						},
					},
					{
						err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 40),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
				},
			},
		},
		{
			name: "test no majority and no dec",
			relayErrors: RelayErrors{
				onFailureMergeAll: true,
				relayErrors: []RelayError{
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 20),
						},
					},
					{
						err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 30),
						},
					},
					{
						err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 40),
						},
					},
					{
						err: fmt.Errorf(expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderQoSExcellenceSummery: sdk.OneDec(),
							ProviderStake:                sdk.NewInt64Coin("ulava", 10),
						},
					},
				},
			},
		},
	}
	for _, te := range testStruct {
		t.Run(te.name, func(t *testing.T) {
			result := te.relayErrors.GetBestErrorMessageForUser()
			require.Equal(t, result.err.Error(), expectedValue)
		})
	}
}
