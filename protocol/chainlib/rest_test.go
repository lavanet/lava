package chainlib

import (
	"testing"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/assert"
)

func TestRestChainParser_Spec(t *testing.T) {
	// create a new instance of RestChainParser
	apip, err := NewRestChainParser()
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

func TestRestChainParser_NilGuard(t *testing.T) {
	var apip *RestChainParser

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

func TestRestGetSupportedApi(t *testing.T) {
	connectionType := "test"
	// Test case 1: Successful scenario, returns a supported API
	apip := &RestChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1"}: {api: &spectypes.Api{Name: "API1", Enabled: true}, collectionKey: CollectionKey{ConnectionType: connectionType}}},
		},
	}
	api, err := apip.getSupportedApi("API1", connectionType)
	assert.NoError(t, err)
	assert.Equal(t, "API1", api.api.Name)

	// Test case 2: Returns error if the API does not exist
	apip = &RestChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1"}: {api: &spectypes.Api{Name: "API1", Enabled: true}, collectionKey: CollectionKey{ConnectionType: connectionType}}},
		},
	}
	_, err = apip.getSupportedApi("API2", connectionType)
	assert.Error(t, err)
	assert.Equal(t, "rest api not supported API2", err.Error())

	// Test case 3: Returns error if the API is disabled
	apip = &RestChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1"}: {api: &spectypes.Api{Name: "API1", Enabled: false}, collectionKey: CollectionKey{ConnectionType: connectionType}}},
		},
	}
	_, err = apip.getSupportedApi("API1", connectionType)
	assert.Error(t, err)
	assert.Equal(t, "api is disabled", err.Error())
}

func TestRestParseMessage(t *testing.T) {
	connectionType := "test"
	apip := &RestChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{
				{Name: "API1", ConnectionType: connectionType}: {api: &spectypes.Api{Name: "API1", Enabled: false}, collectionKey: CollectionKey{ConnectionType: connectionType}},
			},
			apiCollections: map[CollectionKey]*spectypes.ApiCollection{{ConnectionType: connectionType}: {CollectionData: spectypes.CollectionData{ApiInterface: spectypes.APIInterfaceRest}}},
		},
	}

	msg, err := apip.ParseMsg("API1", []byte("test message"), spectypes.APIInterfaceRest, nil)

	assert.Nil(t, err)
	assert.Equal(t, msg.GetApi().Name, apip.serverApis[ApiKey{Name: "API1", ConnectionType: connectionType}].api.Name)

	restMessage := rpcInterfaceMessages.RestMessage{
		Msg:      []byte("test message"),
		Path:     "API1",
		SpecPath: "API1",
	}

	assert.Equal(t, restMessage, msg.GetRPCMessage())
}
