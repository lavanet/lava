package chainlib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	plantypes "github.com/lavanet/lava/v2/x/plans/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, apip.spec.DataReliabilityEnabled, enabled)
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
	apip.getSupportedApi("", "", "")
	apip.ParseMsg("", []byte{}, "", nil, extensionslib.ExtensionInfo{LatestBlock: 0})
}

func TestJSONGetSupportedApi(t *testing.T) {
	// Test case 1: Successful scenario, returns a supported API
	apip := &JsonRPCChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: true}, collectionKey: CollectionKey{ConnectionType: connectionType_test}}},
		},
	}
	api, err := apip.getSupportedApi("API1", connectionType_test, "")
	assert.NoError(t, err)
	assert.Equal(t, "API1", api.api.Name)

	// Test case 2: Returns error if the API does not exist
	apip = &JsonRPCChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: true}, collectionKey: CollectionKey{ConnectionType: connectionType_test}}},
		},
	}
	_, err = apip.getSupportedApi("API2", connectionType_test, "")
	assert.Error(t, err)

	// Test case 3: Returns error if the API is disabled
	apip = &JsonRPCChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: false}, collectionKey: CollectionKey{ConnectionType: connectionType_test}}},
		},
	}
	_, err = apip.getSupportedApi("API1", connectionType_test, "")
	assert.Error(t, err)
}

func TestJSONParseMessage(t *testing.T) {
	apip := &JsonRPCChainParser{
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
			apiCollections: map[CollectionKey]*spectypes.ApiCollection{{ConnectionType: connectionType_test}: {Enabled: true, CollectionData: spectypes.CollectionData{ApiInterface: spectypes.APIInterfaceJsonRPC}}},
		},
	}

	data := rpcInterfaceMessages.JsonrpcMessage{
		Method: "API1",
	}

	marshalledData, _ := json.Marshal(data)

	msg, err := apip.ParseMsg("API1", marshalledData, connectionType_test, nil, extensionslib.ExtensionInfo{LatestBlock: 0})

	assert.Nil(t, err)
	assert.Equal(t, msg.GetApi().Name, apip.serverApis[ApiKey{Name: "API1", ConnectionType: connectionType_test}].api.Name)
	requestedBlock, _ := msg.RequestedBlock()
	assert.Equal(t, requestedBlock, int64(-2))
	assert.Equal(t, msg.GetApiCollection().CollectionData.ApiInterface, spectypes.APIInterfaceJsonRPC)
}

func TestJsonRpcChainProxy(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x10a7a08"}`)
	})

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, serverHandle, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)
	block, err := chainFetcher.FetchLatestBlockNum(ctx)
	require.Greater(t, block, int64(0))
	require.NoError(t, err)
	_, err = chainFetcher.FetchBlockHashByNum(ctx, block)
	errMsg := "GET_BLOCK_BY_NUM Failed ParseMessageResponse {error:invalid parser input format"
	require.True(t, err.Error()[:len(errMsg)] == errMsg, err.Error())
	if closeServer != nil {
		closeServer()
	}
}

func TestAddonAndVerifications(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0xf9ccdff90234a064"}`)
	})

	chainParser, chainRouter, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, serverHandle, "../../", []string{"debug"})
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainRouter)
	require.NotNil(t, chainFetcher)

	verifications, err := chainParser.GetVerifications([]string{"debug"})
	require.NoError(t, err)
	require.NotEmpty(t, verifications)
	for _, verification := range verifications {
		parsing := &verification.ParseDirective
		collectionType := verification.ConnectionType
		chainMessage, err := CraftChainMessage(parsing, collectionType, chainParser, nil, nil)
		require.NoError(t, err)
		reply, _, _, _, _, err := chainRouter.SendNodeMsg(ctx, nil, chainMessage, []string{verification.Extension})
		require.NoError(t, err)
		_, err = FormatResponseForParsing(reply.RelayReply, chainMessage)
		require.NoError(t, err)
	}
	if closeServer != nil {
		closeServer()
	}
}

func TestExtensions(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0xf9ccdff90234a064"}`)
	})

	specname := "ETH1"
	chainParser, chainRouter, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, specname, spectypes.APIInterfaceJsonRPC, serverHandle, "../../", []string{"archive"})
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainRouter)
	require.NotNil(t, chainFetcher)
	configuredExtensions := map[string]struct{}{
		"archive": {},
	}
	spec, err := keepertest.GetASpec(specname, "../../", nil, nil)
	require.NoError(t, err)

	chainParser.SetPolicy(&plantypes.Policy{ChainPolicies: []plantypes.ChainPolicy{{ChainId: specname, Requirements: []plantypes.ChainRequirement{{Collection: spectypes.CollectionData{ApiInterface: "jsonrpc"}, Extensions: []string{"archive"}}}}}}, specname, "jsonrpc")
	parsingForCrafting, collectionData, ok := chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM)
	require.True(t, ok)
	cuCost := uint64(0)
	for _, api := range spec.ApiCollections[0].Apis {
		if api.Name == parsingForCrafting.ApiName {
			cuCost = api.ComputeUnits
			break
		}
	}
	require.NotZero(t, cuCost)
	cuCostExt := uint64(0)
	for _, ext := range spec.ApiCollections[0].Extensions {
		_, ok := configuredExtensions[ext.Name]
		if ok {
			cuCostExt = cuCost * ext.CuMultiplier
			break
		}
	}
	require.NotZero(t, cuCostExt)
	latestTemplate := strings.Replace(parsingForCrafting.FunctionTemplate, "0x%x", "%s", 1)
	latestReq := []byte(fmt.Sprintf(latestTemplate, "latest"))
	reqSpecific := []byte(fmt.Sprintf(parsingForCrafting.FunctionTemplate, 99))
	// with latest block not set
	chainMessage, err := chainParser.ParseMsg("", latestReq, collectionData.Type, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	require.Equal(t, parsingForCrafting.ApiName, chainMessage.GetApi().Name)
	require.Empty(t, chainMessage.GetExtensions())
	require.Equal(t, cuCost, chainMessage.GetApi().ComputeUnits)

	chainMessage, err = chainParser.ParseMsg("", reqSpecific, collectionData.Type, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	require.Equal(t, parsingForCrafting.ApiName, chainMessage.GetApi().Name)
	require.Len(t, chainMessage.GetExtensions(), 1)
	require.Equal(t, "archive", chainMessage.GetExtensions()[0].Name)
	require.Equal(t, cuCostExt, chainMessage.GetApi().ComputeUnits)

	// with latest block set
	chainMessage, err = chainParser.ParseMsg("", latestReq, collectionData.Type, nil, extensionslib.ExtensionInfo{LatestBlock: 100})
	require.NoError(t, err)
	require.Equal(t, parsingForCrafting.ApiName, chainMessage.GetApi().Name)
	require.Empty(t, chainMessage.GetExtensions())
	require.Equal(t, cuCost, chainMessage.GetApi().ComputeUnits)

	chainMessage, err = chainParser.ParseMsg("", reqSpecific, collectionData.Type, nil, extensionslib.ExtensionInfo{LatestBlock: 100})
	require.NoError(t, err)
	require.Equal(t, parsingForCrafting.ApiName, chainMessage.GetApi().Name)
	require.Len(t, chainMessage.GetExtensions(), 1)
	require.Equal(t, "archive", chainMessage.GetExtensions()[0].Name)
	require.Equal(t, cuCostExt, chainMessage.GetApi().ComputeUnits)
	if closeServer != nil {
		closeServer()
	}
}

func TestJsonRpcBatchCall(t *testing.T) {
	ctx := context.Background()
	gotCalled := false
	const response = `[{"jsonrpc":"2.0","id":1,"result":"0x1"},{"jsonrpc":"2.0","id":2,"result":[]},{"jsonrpc":"2.0","id":3,"result":"0x114b56b"}]`
	batchCallData := `[{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]},{"jsonrpc":"2.0","id":2,"method":"eth_accounts","params":[]},{"jsonrpc":"2.0","id":3,"method":"eth_blockNumber","params":[]}]`
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

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, serverHandle, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)

	chainMessage, err := chainParser.ParseMsg("", []byte(batchCallData), http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	requestedBlock, _ := chainMessage.RequestedBlock()
	require.Equal(t, spectypes.LATEST_BLOCK, requestedBlock)
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

func TestJsonRpcBatchCallSameID(t *testing.T) {
	ctx := context.Background()
	gotCalled := false
	batchCallData := `[{"jsonrpc":"2.0","id":1,"method":"eth_chainId"},{"jsonrpc":"2.0","id":1,"method":"eth_chainId"}]` // call same id
	const responseExpected = `[{"jsonrpc":"2.0","id":1,"result":"0x1"},{"jsonrpc":"2.0","id":1,"result":"0x1"}]`         // response is expected to be like the user asked
	// we are sending and receiving something else
	const response = `[{"jsonrpc":"2.0","id":1,"result":"0x1"},{"jsonrpc":"2.0","id":3,"result":"0x1"}]`                     // response of the server is to the different ids
	sentBatchCallData := `[{"jsonrpc":"2.0","id":1,"method":"eth_chainId"},{"jsonrpc":"2.0","id":3,"method":"eth_chainId"}]` // what is being sent is different ids
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCalled = true
		data := make([]byte, len([]byte(batchCallData)))
		r.Body.Read(data)
		// require.NoError(t, err)
		require.Equal(t, sentBatchCallData, string(data))
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, response)
	})

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, serverHandle, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)

	chainMessage, err := chainParser.ParseMsg("", []byte(batchCallData), http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	requestedBlock, _ := chainMessage.RequestedBlock()
	require.Equal(t, spectypes.LATEST_BLOCK, requestedBlock)
	relayReply, _, _, _, _, err := chainProxy.SendNodeMsg(ctx, nil, chainMessage, nil)
	require.True(t, gotCalled)
	require.NoError(t, err)
	require.NotNil(t, relayReply)
	require.Equal(t, responseExpected, string(relayReply.RelayReply.Data))
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()
}

func TestJsonRpcInternalPathsMultipleVersions(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"%s"}`, r.RequestURI)
	})
	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "STRK", spectypes.APIInterfaceJsonRPC, serverHandle, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)
	v5_path := "/rpc/v0_5"
	v6_path := "/rpc/v0_6"
	req_data := []byte(`{"jsonrpc": "2.0", "id": 1, "method": "starknet_specVersion", "params": []}`)
	chainMessage, err := chainParser.ParseMsg("", req_data, http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	api := chainMessage.GetApi()
	collection := chainMessage.GetApiCollection()
	require.Equal(t, "starknet_specVersion", api.Name)
	require.Equal(t, "", collection.CollectionData.InternalPath)

	chainMessage, err = chainParser.ParseMsg(v5_path, req_data, http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	api = chainMessage.GetApi()
	collection = chainMessage.GetApiCollection()
	require.Equal(t, "starknet_specVersion", api.Name)
	require.Equal(t, v5_path, collection.CollectionData.InternalPath)

	chainMessage, err = chainParser.ParseMsg(v6_path, req_data, http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	api = chainMessage.GetApi()
	collection = chainMessage.GetApiCollection()
	require.Equal(t, "starknet_specVersion", api.Name)
	require.Equal(t, v6_path, collection.CollectionData.InternalPath)
	if closeServer != nil {
		closeServer()
	}
}
