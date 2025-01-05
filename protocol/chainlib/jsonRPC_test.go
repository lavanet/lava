package chainlib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	specutils "github.com/lavanet/lava/v4/utils/keeper"
	plantypes "github.com/lavanet/lava/v4/x/plans/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createWebSocketHandler(handler func(string) string) http.HandlerFunc {
	upGrader := websocket.Upgrader{}

	// Create a simple websocket server that mocks the node
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upGrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			panic("got error in upgrader")
		}
		defer conn.Close()

		for {
			// Read the request
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				panic("got error in ReadMessage")
			}
			fmt.Println("got ws message", string(message), messageType)
			retMsg := handler(string(message))
			conn.WriteMessage(messageType, []byte(retMsg))
			fmt.Println("writing ws message", string(message), messageType)
		}
	}
}

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
	apiCont, err := apip.getSupportedApi("API2", connectionType_test, "")
	if err == nil {
		assert.Equal(t, "Default-API2", apiCont.api.Name)
	} else {
		assert.ErrorIs(t, err, common.APINotSupportedError)
	}

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
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x10a7a08"}`)
	})

	wsServerHandler := func(message string) string {
		return `{"jsonrpc":"2.0","id":1,"result":"0x10a7a08"}`
	}

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, serverHandler, createWebSocketHandler(wsServerHandler), "../../", nil)
	if closeServer != nil {
		defer closeServer()
	}

	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)

	block, err := chainFetcher.FetchLatestBlockNum(ctx)
	require.Greater(t, block, int64(0))
	require.NoError(t, err)

	_, err = chainFetcher.FetchBlockHashByNum(ctx, block)
	expectedErrMsg := "GET_BLOCK_BY_NUM Failed ParseMessageResponse {error:failed to parse with legacy block parser ErrMsg: blockParsing -"
	actualErrMsg := err.Error()[:len(expectedErrMsg)]
	require.Equal(t, expectedErrMsg, actualErrMsg, err.Error())
}

func TestAddonAndVerifications(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0xf9ccdff90234a064"}`)
	})

	wsServerHandler := func(message string) string {
		return `{"jsonrpc":"2.0","id":1,"result":"0xf9ccdff90234a064"}`
	}

	chainParser, chainRouter, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, serverHandle, createWebSocketHandler(wsServerHandler), "../../", []string{"debug"})
	if closeServer != nil {
		defer closeServer()
	}

	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainRouter)
	require.NotNil(t, chainFetcher)

	verifications, err := chainParser.GetVerifications([]string{"debug"}, "", "jsonrpc")
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
}

func TestExtensions(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0xf9ccdff90234a064"}`)
	})

	wsServerHandler := func(message string) string {
		return `{"jsonrpc":"2.0","id":1,"result":"0xf9ccdff90234a064"}`
	}

	specname := "ETH1"
	chainParser, chainRouter, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, specname, spectypes.APIInterfaceJsonRPC, serverHandle, createWebSocketHandler(wsServerHandler), "../../", []string{"archive"})
	if closeServer != nil {
		defer closeServer()
	}

	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainRouter)
	require.NotNil(t, chainFetcher)
	configuredExtensions := map[string]struct{}{
		"archive": {},
	}
	spec, err := specutils.GetASpec(specname, "../../", nil, nil)
	require.NoError(t, err)

	chainParser.SetPolicy(&plantypes.Policy{ChainPolicies: []plantypes.ChainPolicy{{ChainId: specname, Requirements: []plantypes.ChainRequirement{{Collection: spectypes.CollectionData{ApiInterface: "jsonrpc"}, Extensions: []string{"archive"}}}}}}, specname, "jsonrpc")
	parsingForCrafting, apiCollection, ok := chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM)
	require.True(t, ok)
	collectionData := apiCollection.CollectionData
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

	wsServerHandler := func(message string) string {
		require.Equal(t, batchCallData, message)
		return response
	}

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, serverHandle, createWebSocketHandler(wsServerHandler), "../../", nil)
	if closeServer != nil {
		defer closeServer()
	}

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

	wsServerHandler := func(message string) string {
		require.Equal(t, sentBatchCallData, message)
		return response
	}

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, serverHandle, createWebSocketHandler(wsServerHandler), "../../", nil)
	if closeServer != nil {
		defer closeServer()
	}

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
}

func TestJsonRpcInternalPathsMultipleVersionsStarkNet(t *testing.T) {
	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"%s"}`, r.RequestURI)
	})

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "STRK", spectypes.APIInterfaceJsonRPC, serverHandle, nil, "../../", nil)
	if closeServer != nil {
		defer closeServer()
	}

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
}

func TestJsonRpcInternalPathsMultipleVersionsAvalanche(t *testing.T) {
	type reqWithApiName struct {
		apiName string
		reqData []byte
	}

	// TODO: Add the empty path back in once the ETH spec will be fixed
	// allPaths := []string{"", "/C/rpc", "/C/avax", "/P", "/X"}
	allPaths := []string{"/C/rpc", "/C/avax", "/P", "/X"}
	pathToReqData := map[string]reqWithApiName{
		"/C/rpc": { // Eth jsonrpc path
			apiName: "eth_blockNumber",
			reqData: []byte(`{"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber", "params": []}`),
		},
		"/C/avax": { // Avalanche jsonrpc path
			apiName: "avax.export",
			reqData: []byte(`{"jsonrpc": "2.0", "id": 1, "method": "avax.export", "params": []}`),
		},
		"/P": { // Platform jsonrpc path
			apiName: "platform.addDelegator",
			reqData: []byte(`{"jsonrpc": "2.0", "id": 1, "method": "platform.addDelegator", "params": []}`),
		},
		"/X": { // Avm jsonrpc path
			apiName: "avm.getAssetDescription",
			reqData: []byte(`{"jsonrpc": "2.0", "id": 1, "method": "avm.getAssetDescription", "params": []}`),
		},
	}

	ctx := context.Background()
	serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"%s"}`, r.RequestURI)
	})

	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "AVAX", spectypes.APIInterfaceJsonRPC, serverHandle, nil, "../../", nil)
	if closeServer != nil {
		defer closeServer()
	}

	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)

	for correctPath, reqDataWithApiName := range pathToReqData {
		for _, path := range allPaths {
			shouldErr := path != correctPath
			t.Run(fmt.Sprintf("ApiName:%s,CorrectPath:%s,Path:%s,ShouldError:%v", reqDataWithApiName.apiName, correctPath, path, shouldErr), func(t *testing.T) {
				chainMessage, err := chainParser.ParseMsg(path, reqDataWithApiName.reqData, http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})

				if !shouldErr {
					require.NoError(t, err)
					api := chainMessage.GetApi()
					collection := chainMessage.GetApiCollection()
					require.Equal(t, reqDataWithApiName.apiName, api.Name)
					require.Equal(t, correctPath, collection.CollectionData.InternalPath)
				} else {
					if err == nil {
						require.Contains(t, chainMessage.GetApi().Name, "Default-")
					} else {
						require.ErrorIs(t, err, common.APINotSupportedError)
						require.Nil(t, chainMessage)
					}
				}
			})
		}
	}
}

func TestJsonRPC_SpecUpdateWithAddons(t *testing.T) {
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
		ApiCollections: []*spectypes.ApiCollection{
			{
				Enabled: true,
				CollectionData: spectypes.CollectionData{
					ApiInterface: "jsonrpc",
					InternalPath: "",
					Type:         "POST",
					AddOn:        "debug",
				},
				Apis: []*spectypes.Api{
					{
						Enabled: true,
						Name:    "foo",
					},
				},
			},
		},
	}

	// Set the spec for the first time
	apip.SetSpec(spec)

	// At first, addon should be disabled
	require.False(t, apip.allowedAddons["debug"])

	// Setting the spec again, for sanity check
	apip.SetSpec(spec)

	// Sanity check that addon still disabled
	require.False(t, apip.allowedAddons["debug"])

	// Allow the addon
	apip.SetPolicyFromAddonAndExtensionMap(map[string]struct{}{
		"debug": {},
	})

	// Sanity check
	require.True(t, apip.allowedAddons["debug"])

	// Set the spec again
	apip.SetSpec(spec)

	// Should stay the same
	require.True(t, apip.allowedAddons["debug"])

	// Disallow the addon
	apip.SetPolicyFromAddonAndExtensionMap(map[string]struct{}{})

	// Sanity check
	require.False(t, apip.allowedAddons["debug"])

	// Set the spec again
	apip.SetSpec(spec)

	// Should stay the same
	require.False(t, apip.allowedAddons["debug"])
}

func TestJsonRPC_SpecUpdateWithExtensions(t *testing.T) {
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
		ApiCollections: []*spectypes.ApiCollection{
			{
				Enabled: true,
				CollectionData: spectypes.CollectionData{
					ApiInterface: "jsonrpc",
					InternalPath: "",
					Type:         "POST",
					AddOn:        "",
				},
				Extensions: []*spectypes.Extension{
					{
						Name: "archive",
						Rule: &spectypes.Rule{
							Block: 123,
						},
					},
				},
			},
		},
	}

	extensionKey := extensionslib.ExtensionKey{
		Extension:      "archive",
		ConnectionType: "POST",
		InternalPath:   "",
		Addon:          "",
	}

	isExtensionConfigured := func() bool {
		_, isConfigured := apip.extensionParser.GetConfiguredExtensions()[extensionKey]
		return isConfigured
	}

	// Set the spec for the first time
	apip.SetSpec(spec)

	// At first, extension should not be configured
	require.False(t, isExtensionConfigured())

	// Setting the spec again, for sanity check
	apip.SetSpec(spec)

	// Sanity check that extension is still not configured
	require.False(t, isExtensionConfigured())

	// Allow the extension
	apip.SetPolicyFromAddonAndExtensionMap(map[string]struct{}{
		"archive": {},
	})

	// Sanity check
	require.True(t, isExtensionConfigured())

	// Set the spec again
	apip.SetSpec(spec)

	// Should stay the same
	require.True(t, isExtensionConfigured())

	// Disallow the extension
	apip.SetPolicyFromAddonAndExtensionMap(map[string]struct{}{})

	// Sanity check
	require.False(t, isExtensionConfigured())

	// Set the spec again
	apip.SetSpec(spec)

	// Should stay the same
	require.False(t, isExtensionConfigured())
}
