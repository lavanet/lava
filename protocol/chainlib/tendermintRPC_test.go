package chainlib

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/assert"
)

func TestTendermintChainParser_Spec(t *testing.T) {
	// create a new instance of RestChainParser
	apip, err := NewTendermintRpcChainParser()
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

func TestTendermintChainParser_NilGuard(t *testing.T) {
	var apip *TendermintChainParser

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("apip methods missing nill guard, panicked with: %v", r)
		}
	}()

	apip.SetSpec(spectypes.Spec{})
	apip.DataReliabilityParams()
	apip.ChainBlockStats()
	apip.getSupportedApi("", "")
	apip.ParseMsg("", []byte{}, "", nil)
}

func TestTendermintGetSupportedApi(t *testing.T) {
	// Test case 1: Successful scenario, returns a supported API
	apip := &TendermintChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: true}, collectionKey: CollectionKey{ConnectionType: connectionType_test}}},
		},
	}
	api, err := apip.getSupportedApi("API1", connectionType_test)
	assert.NoError(t, err)
	assert.Equal(t, "API1", api.api.Name)

	// Test case 2: Returns error if the API does not exist
	apip = &TendermintChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: true}, collectionKey: CollectionKey{ConnectionType: connectionType_test}}},
		},
	}
	_, err = apip.getSupportedApi("API2", connectionType_test)
	assert.Error(t, err)

	// Test case 3: Returns error if the API is disabled
	apip = &TendermintChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: false}, collectionKey: CollectionKey{ConnectionType: connectionType_test}}},
		},
	}
	_, err = apip.getSupportedApi("API1", connectionType_test)
	assert.Error(t, err)
}

func TestTendermintParseMessage(t *testing.T) {
	apip := &TendermintChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{
				{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{
					Name:    "API1",
					Enabled: true,
					BlockParsing: spectypes.BlockParser{
						ParserArg:  []string{"latest"},
						ParserFunc: spectypes.PARSER_FUNC_DEFAULT,
					},
				}, collectionKey: CollectionKey{ConnectionType: connectionType_test}},
			},
			apiCollections: map[CollectionKey]*spectypes.ApiCollection{{ConnectionType: connectionType_test}: {Enabled: true, CollectionData: spectypes.CollectionData{ApiInterface: spectypes.APIInterfaceTendermintRPC}}},
		},
	}

	data := rpcInterfaceMessages.TendermintrpcMessage{
		JsonrpcMessage: rpcInterfaceMessages.JsonrpcMessage{
			Method: "API1",
		},
		Path: "",
	}

	marshalledData, _ := json.Marshal(data)

	msg, err := apip.ParseMsg("API1", marshalledData, connectionType_test, nil)

	assert.Nil(t, err)
	assert.Equal(t, msg.GetApi().Name, apip.serverApis[ApiKey{Name: "API1", ConnectionType: connectionType_test}].api.Name)
	assert.Equal(t, msg.RequestedBlock(), int64(-2))
}
