package chainlib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, apip.spec.DataReliabilityEnabled, enabled)
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
	apip.ParseMsg("", []byte{}, "", nil, extensionslib.ExtensionInfo{LatestBlock: 0})
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
	apiCont, err := apip.getSupportedApi("API2", connectionType_test)
	if err == nil {
		assert.Equal(t, "Default-API2", apiCont.api.Name)
	} else {
		assert.ErrorIs(t, err, common.APINotSupportedError)
	}

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

	msg, err := apip.ParseMsg("API1", marshalledData, connectionType_test, nil, extensionslib.ExtensionInfo{LatestBlock: 0})

	assert.Nil(t, err)
	assert.Equal(t, msg.GetApi().Name, apip.serverApis[ApiKey{Name: "API1", ConnectionType: connectionType_test}].api.Name)
	requestedBlock, _ := msg.RequestedBlock()
	assert.Equal(t, requestedBlock, int64(-2))
}

func TestTendermintRpcChainProxy(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"jsonrpc": "2.0",
			"id": 1,"result": {
				"sync_info": {
					"latest_block_height": "1947"
				},
				"block_id": {
					"hash": "ABABABABABABABABABABABAB"
				}
			}
		}`)
	})

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceTendermintRPC, serverHandle, nil, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)
	block, err := chainFetcher.FetchLatestBlockNum(ctx)
	require.Greater(t, block, int64(0))
	require.NoError(t, err)
	_, err = chainFetcher.FetchBlockHashByNum(ctx, block)
	require.NoError(t, err)
	if closeServer != nil {
		closeServer()
	}
}

func TestTendermintRpcBatchCall(t *testing.T) {
	ctx := context.Background()
	gotCalled := false
	batchCallData := `[{"jsonrpc":"2.0","id":1,"method":"block","params":{"height":"99"}},{"jsonrpc":"2.0","id":2,"method":"block","params":{"height":"100"}}]`
	const response = `[{"jsonrpc":"2.0","id":1,"result":{"block_id":{},"block":{}}},{"jsonrpc":"2.0","id":2,"result":{"block_id":{},"block":{}}}]`
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCalled = true
		data := make([]byte, len([]byte(batchCallData)))
		r.Body.Read(data)
		// require.NoError(t, err)
		require.Equal(t, batchCallData, string(data))
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, response)
	})

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceTendermintRPC, serverHandle, nil, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)

	chainMessage, err := chainParser.ParseMsg("", []byte(batchCallData), "", nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	requestedBlock, earliestReqBlock := chainMessage.RequestedBlock()
	require.Equal(t, int64(100), requestedBlock)
	require.Equal(t, int64(99), earliestReqBlock)
	relayReply, _, _, _, _, err := chainProxy.SendNodeMsg(ctx, nil, chainMessage, nil)
	require.True(t, gotCalled)
	require.NoError(t, err)
	require.NotNil(t, relayReply)
	require.Equal(t, response, string(relayReply.RelayReply.Data))
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()
}

func TestTendermintRpcBatchCallWithSameID(t *testing.T) {
	ctx := context.Background()
	gotCalled := false
	batchCallData := `[{"jsonrpc":"2.0","id":1,"method":"block","params":{"height":"99"}},{"jsonrpc":"2.0","id":1,"method":"block","params":{"height":"100"}}]`
	const response = `[{"jsonrpc":"2.0","id":1,"result":{"block_id1111111111":{},"block":{}}},{"jsonrpc":"2.0","id":1,"result":{"block_id222222":{},"block":{}}}]`

	const nodeResponse = `[{"jsonrpc":"2.0","id":1,"result":{"block_id1111111111":{},"block":{}}},{"jsonrpc":"2.0","id":2,"result":{"block_id222222":{},"block":{}}}]`
	nodeBatchCallData := `[{"jsonrpc":"2.0","id":1,"method":"block","params":{"height":"99"}},{"jsonrpc":"2.0","id":2,"method":"block","params":{"height":"100"}}]`
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCalled = true
		data := make([]byte, len([]byte(nodeBatchCallData)))
		r.Body.Read(data)
		// require.NoError(t, err)
		require.Equal(t, nodeBatchCallData, string(data))
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, nodeResponse)
	})

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceTendermintRPC, serverHandle, nil, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)

	chainMessage, err := chainParser.ParseMsg("", []byte(batchCallData), "", nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	requestedBlock, earliestReqBlock := chainMessage.RequestedBlock()
	require.Equal(t, int64(100), requestedBlock)
	require.Equal(t, int64(99), earliestReqBlock)
	relayReply, _, _, _, _, err := chainProxy.SendNodeMsg(ctx, nil, chainMessage, nil)
	require.True(t, gotCalled)
	require.NoError(t, err)
	require.NotNil(t, relayReply)
	require.Equal(t, response, string(relayReply.RelayReply.Data))
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()
}

func TestTendermintURIRPC(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"jsonrpc": "2.0",
			"id": 1,"result": "ok"
		}`)
	})

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceTendermintRPC, serverHandle, nil, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)
	requestUrl := "tx_search?query=%22recv_packet.packet_src_channel=%27channel-227%27%20AND%20recv_packet.packet_sequence=%271123%27%20%20AND%20recv_packet.packet_dst_channel=%27channel-3%27%22"
	chainMessage, err := chainParser.ParseMsg(requestUrl, nil, "", nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	nodeMessage, ok := chainMessage.GetRPCMessage().(*rpcInterfaceMessages.TendermintrpcMessage)
	require.True(t, ok)
	params := nodeMessage.GetParams()
	casted, ok := params.(map[string]interface{})
	require.True(t, ok)
	_, ok = casted["query"]
	require.True(t, ok)
	if closeServer != nil {
		closeServer()
	}
}
