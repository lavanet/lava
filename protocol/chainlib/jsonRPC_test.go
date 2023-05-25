package chainlib

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/assert"
)

func TestJSONChainParser_Spec(t *testing.T) {
	// create a new instance of RestChainParser
	apip, err := NewJrpcChainParser()
	if err != nil {
		t.Errorf("Error creating RestChainParser: %v", err)
	}

	// set the spec
	spec := spectypes.Spec{
		Enabled:                       true,
		ReliabilityThreshold:          10,
		AllowedBlockLagForQosSync:     11,
		AverageBlockTime:              12000,
		BlockDistanceForFinalizedData: 13,
		BlocksInFinalizationProof:     14,
	}
	apip.SetSpec(spec)

	// fetch data reliability params
	enabled, dataReliabilityThreshold := apip.DataReliabilityParams()

	// fetch chain block stats
	allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData, blocksInFinalizationProof := apip.ChainBlockStats()

	// convert block time
	AverageBlockTime := time.Duration(apip.spec.AverageBlockTime) * time.Millisecond

	// check that the spec was set correctly
	assert.Equal(t, apip.spec.Enabled, enabled)
	assert.Equal(t, apip.spec.GetReliabilityThreshold(), dataReliabilityThreshold)
	assert.Equal(t, apip.spec.AllowedBlockLagForQosSync, allowedBlockLagForQosSync)
	assert.Equal(t, apip.spec.BlockDistanceForFinalizedData, blockDistanceForFinalizedData)
	assert.Equal(t, apip.spec.BlocksInFinalizationProof, blocksInFinalizationProof)
	assert.Equal(t, AverageBlockTime, averageBlockTime)
}

func TestJSONChainParser_NilGuard(t *testing.T) {
	var apip *JsonRPCChainParser

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("apip methods missing nill guard, panicked with: %v", r)
		}
	}()

	apip.SetSpec(spectypes.Spec{})
	apip.DataReliabilityParams()
	apip.ChainBlockStats()
	apip.getSupportedApi("", "")
	apip.ParseMsg("", []byte{}, "")
}

func TestJSONGetSupportedApi(t *testing.T) {
	connectionType := "test"
	// Test case 1: Successful scenario, returns a supported API
	apip := &JsonRPCChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]*spectypes.Api{ApiKey{Name: "API1", CollectionKey: CollectionKey{ConnectionType: connectionType}}: {Name: "API1", Enabled: true}},
		},
	}
	api, err := apip.getSupportedApi("API1", connectionType)
	assert.NoError(t, err)
	assert.Equal(t, "API1", api.Name)

	// Test case 2: Returns error if the API does not exist
	apip = &JsonRPCChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]*spectypes.Api{ApiKey{Name: "API1", CollectionKey: CollectionKey{ConnectionType: connectionType}}: {Name: "API1", Enabled: true}},
		},
	}
	_, err = apip.getSupportedApi("API2", connectionType)
	assert.Error(t, err)
	assert.Equal(t, "api not supported", err.Error())

	// Test case 3: Returns error if the API is disabled
	apip = &JsonRPCChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]*spectypes.Api{ApiKey{Name: "API1", CollectionKey: CollectionKey{ConnectionType: connectionType}}: {Name: "API1", Enabled: false}},
		},
	}
	_, err = apip.getSupportedApi("API1", connectionType)
	assert.Error(t, err)
	assert.Equal(t, "api is disabled", err.Error())
}

func TestJSONParseMessage(t *testing.T) {
	connectionType := "test"
	apip := &JsonRPCChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]*spectypes.Api{
				ApiKey{Name: "API1", CollectionKey: CollectionKey{ConnectionType: connectionType}}: {
					Name:    "API1",
					Enabled: true,
				},
			},
			apiCollections: map[CollectionKey]*spectypes.ApiCollection{{ConnectionType: connectionType}: {CollectionData: spectypes.CollectionData{ApiInterface: spectypes.APIInterfaceJsonRPC}}},
		},
	}

	data := rpcInterfaceMessages.JsonrpcMessage{
		Method: "API1",
	}

	marshalledData, _ := json.Marshal(data)

	msg, err := apip.ParseMsg("API1", marshalledData, spectypes.APIInterfaceJsonRPC)

	assert.Nil(t, err)
	assert.Equal(t, msg.GetApi().Name, apip.serverApis[ApiKey{Name: "API1", CollectionKey: CollectionKey{ConnectionType: connectionType}}].Name)
	assert.Equal(t, msg.RequestedBlock(), int64(-2))
	assert.Equal(t, msg.GetApiCollection().CollectionData.ApiInterface, spectypes.APIInterfaceJsonRPC)
}
