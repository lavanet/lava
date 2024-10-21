package chainlib

import (
	"fmt"
	"net/http"
	"testing"

	testcommon "github.com/lavanet/lava/v4/testutil/common"
	"github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestCraftChainMessage(t *testing.T) {
	type play struct {
		apiInterface string
		craftData    *CraftData
	}

	expectedInternalPath := "/x"
	method := "banana"

	playBook := []play{
		{
			apiInterface: types.APIInterfaceJsonRPC,
			craftData: &CraftData{
				Path:           method,
				Data:           []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":[],"id":1}`, method)),
				ConnectionType: http.MethodPost,
				InternalPath:   expectedInternalPath,
			},
		},
		{
			apiInterface: types.APIInterfaceTendermintRPC,
			craftData: &CraftData{
				Path:           method,
				Data:           []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":[],"id":1}`, method)),
				ConnectionType: "",
				InternalPath:   expectedInternalPath,
			},
		},
		{
			apiInterface: types.APIInterfaceRest,
			craftData: &CraftData{
				Data:           []byte(method),
				ConnectionType: http.MethodGet,
				InternalPath:   expectedInternalPath,
			},
		},
		{
			apiInterface: types.APIInterfaceRest,
			craftData: &CraftData{
				Path:           method,
				Data:           []byte(`{"data":"banana"}`),
				ConnectionType: http.MethodPost,
				InternalPath:   expectedInternalPath,
			},
		},
		{
			apiInterface: types.APIInterfaceGrpc,
			craftData: &CraftData{
				Path:           method,
				ConnectionType: "",
				InternalPath:   expectedInternalPath,
			},
		},
	}

	for _, play := range playBook {
		runName := play.apiInterface
		if play.craftData.ConnectionType != "" {
			runName += "_" + play.craftData.ConnectionType
		}

		t.Run(runName, func(t *testing.T) {
			chainParser, err := NewChainParser(play.apiInterface)
			require.NoError(t, err)

			spec := testcommon.CreateMockSpec()
			spec.ApiCollections = []*types.ApiCollection{
				{
					Enabled: true,
					CollectionData: types.CollectionData{
						ApiInterface: play.apiInterface,
						Type:         play.craftData.ConnectionType,
						InternalPath: expectedInternalPath,
					},
					Apis: []*types.Api{
						{
							Name:         method,
							ComputeUnits: 100,
							Enabled:      true,
						},
					},
				},
			}
			chainParser.SetSpec(spec)

			chainMsg, err := CraftChainMessage(&types.ParseDirective{ApiName: method}, play.craftData.ConnectionType, chainParser, play.craftData, nil)
			require.NoError(t, err)
			require.NotNil(t, chainMsg)

			internalPath := chainMsg.GetApiCollection().CollectionData.InternalPath
			require.Equal(t, expectedInternalPath, internalPath)
		})
	}
}
