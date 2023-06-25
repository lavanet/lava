package chainlib

import (
	"sync"
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
	assert.Equal(t, apip.spec.DataReliabilityEnabled, enabled)
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
	apip.getSupportedApi("")
	apip.ParseMsg("", []byte{}, "", nil)
}

func TestRestGetSupportedApi(t *testing.T) {
	// Test case 1: Successful scenario, returns a supported API
	apip := &RestChainParser{
		rwLock:     sync.RWMutex{},
		serverApis: map[string]spectypes.ServiceApi{"API1": {Name: "API1", Enabled: true}},
	}
	api, err := apip.getSupportedApi("API1")
	assert.NoError(t, err)
	assert.Equal(t, "API1", api.Name)

	// Test case 2: Returns error if the API does not exist
	apip = &RestChainParser{
		rwLock:     sync.RWMutex{},
		serverApis: map[string]spectypes.ServiceApi{"API1": {Name: "API1", Enabled: true}},
	}
	_, err = apip.getSupportedApi("API2")
	assert.Error(t, err)
	assert.Equal(t, "rest api not supported API2", err.Error())

	// Test case 3: Returns error if the API is disabled
	apip = &RestChainParser{
		rwLock:     sync.RWMutex{},
		serverApis: map[string]spectypes.ServiceApi{"API1": {Name: "API1", Enabled: false}},
	}
	_, err = apip.getSupportedApi("API1")
	assert.Error(t, err)
	assert.Equal(t, "api is disabled", err.Error())
}

func TestRestParseMessage(t *testing.T) {
	apip := &RestChainParser{
		rwLock: sync.RWMutex{},
		serverApis: map[string]spectypes.ServiceApi{
			"API1": {
				Name:    "API1",
				Enabled: true,
				ApiInterfaces: []spectypes.ApiInterface{{
					Type: spectypes.APIInterfaceRest,
				}},
			},
		},
	}

	msg, err := apip.ParseMsg("API1", []byte("test message"), spectypes.APIInterfaceRest, nil)

	assert.Nil(t, err)
	assert.Equal(t, msg.GetServiceApi().Name, apip.serverApis["API1"].Name)

	restMessage := rpcInterfaceMessages.RestMessage{
		Msg:      []byte("test message"),
		Path:     "API1",
		SpecPath: "API1",
	}

	assert.Equal(t, restMessage, msg.GetRPCMessage())
}
