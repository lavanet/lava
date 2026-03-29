package relaycore

import (
	"fmt"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
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
				OnFailureMergeAll: true,
				RelayErrors: []RelayError{
					{
						Err: fmt.Errorf("test1"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("test2"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             20,
						},
					},
					{
						Err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             30,
						},
					},
					{
						Err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             40,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             50,
						},
					},
				},
			},
		},
		{
			name: "test qos majority error reply",
			relayErrors: RelayErrors{
				OnFailureMergeAll: true,
				RelayErrors: []RelayError{
					{
						Err: fmt.Errorf("test1"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.5,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("test1"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.25,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.6,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.7,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.7,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.7,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.8,
							ProviderStake:             10,
						},
					},
				},
			},
		},
		{
			name: "test text majority over score majority",
			relayErrors: RelayErrors{
				OnFailureMergeAll: true,
				RelayErrors: []RelayError{
					{
						Err: fmt.Errorf("test1"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             1000,
						},
					},
					{
						Err: fmt.Errorf("test2"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             1000,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.0,
							ProviderStake:             0,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.0,
							ProviderStake:             0,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 0.0,
							ProviderStake:             0,
						},
					},
				},
			},
		},
		{
			name: "test majority of error body",
			relayErrors: RelayErrors{
				OnFailureMergeAll: true,
				RelayErrors: []RelayError{
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             20,
						},
					},
					{
						Err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             30,
						},
					},
					{
						Err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             40,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             10,
						},
					},
				},
			},
		},
		{
			name: "test no majority and no dec",
			relayErrors: RelayErrors{
				OnFailureMergeAll: true,
				RelayErrors: []RelayError{
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             10,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             20,
						},
					},
					{
						Err: fmt.Errorf("test3"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             30,
						},
					},
					{
						Err: fmt.Errorf("test4"),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             40,
						},
					},
					{
						Err: fmt.Errorf("%s", expectedValue),
						ProviderInfo: common.ProviderInfo{
							ProviderReputationSummary: 1.0,
							ProviderStake:             10,
						},
					},
				},
			},
		},
	}
	for _, te := range testStruct {
		t.Run(te.name, func(t *testing.T) {
			result := te.relayErrors.GetBestErrorMessageForUser()
			require.Equal(t, result.Err.Error(), expectedValue)
		})
	}
}
