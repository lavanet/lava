package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenesis_Validate(t *testing.T) {
	type test struct {
		Genesis  GenesisState
		ExpError string
	}

	tests := map[string]test{
		"valid": {
			Genesis: GenesisState{
				Params: DefaultParams(),
				Downtimes: []*Downtime{
					{
						Duration: 1,
						Block:    1,
					},
				},
				DowntimesGarbageCollection: []*DowntimeGarbageCollection{
					{
						Block:   1,
						GcBlock: 1,
					},
				},
			},
		},
		"invalid params": {
			Genesis: GenesisState{
				Params: Params{
					DowntimeDuration: -1,
				},
			},
			ExpError: "invalid downtime duration",
		},
		"invalid downtime - duration": {
			Genesis: GenesisState{
				Downtimes: []*Downtime{
					{
						Duration: 0,
					},
				},
			},
			ExpError: "invalid downtime duration",
		},
		"invalid downtime - block": {
			Genesis: GenesisState{
				Params: DefaultParams(),
				Downtimes: []*Downtime{
					{
						Duration: 1 * time.Second,
						Block:    0,
					},
				},
			},
			ExpError: "invalid downtime block",
		},
		"invalid downtime garbage collection": {
			Genesis: GenesisState{
				Params: DefaultParams(),
				DowntimesGarbageCollection: []*DowntimeGarbageCollection{
					{
						Block: 0,
					},
				},
			},
			ExpError: "invalid downtime garbage collection block",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.Genesis.Validate()
			if tc.ExpError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.ExpError)
			}
		})
	}
}
