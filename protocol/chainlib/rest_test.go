package chainlib

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/parser"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	apip.getSupportedApi("", "")
	apip.ParseMsg("", []byte{}, "", nil, extensionslib.ExtensionInfo{LatestBlock: 0})
}

func TestRestGetSupportedApi(t *testing.T) {
	// Test case 1: Successful scenario, returns a supported API
	apip := &RestChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: true}, collectionKey: CollectionKey{ConnectionType: connectionType_test}}},
		},
	}
	api, err := apip.getSupportedApi("API1", connectionType_test)
	assert.NoError(t, err)
	assert.Equal(t, "API1", api.api.Name)

	// Test case 2: Returns error if the API does not exist
	apip = &RestChainParser{
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
	apip = &RestChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: false}, collectionKey: CollectionKey{ConnectionType: connectionType_test}}},
		},
	}
	_, err = apip.getSupportedApi("API1", connectionType_test)
	assert.Error(t, err)
	assert.Equal(t, "api is disabled", err.Error())
}

func TestRestParseMessage(t *testing.T) {
	apip := &RestChainParser{
		BaseChainParser: BaseChainParser{
			serverApis: map[ApiKey]ApiContainer{
				{Name: "API1", ConnectionType: connectionType_test}: {api: &spectypes.Api{Name: "API1", Enabled: true}, collectionKey: CollectionKey{ConnectionType: connectionType_test}},
			},
			apiCollections: map[CollectionKey]*spectypes.ApiCollection{{ConnectionType: connectionType_test}: {Enabled: true, CollectionData: spectypes.CollectionData{ApiInterface: spectypes.APIInterfaceRest}}},
		},
	}

	msg, err := apip.ParseMsg("API1", []byte("test message"), connectionType_test, nil, extensionslib.ExtensionInfo{LatestBlock: 0})

	assert.Nil(t, err)
	assert.Equal(t, msg.GetApi().Name, apip.serverApis[ApiKey{Name: "API1", ConnectionType: connectionType_test}].api.Name)

	restMessage := rpcInterfaceMessages.RestMessage{
		Msg:         []byte("test message"),
		Path:        "API1",
		SpecPath:    "API1",
		BaseMessage: chainproxy.BaseMessage{Headers: []pairingtypes.Metadata{}},
	}

	assert.Equal(t, &restMessage, msg.GetRPCMessage())
}

func TestRestChainProxy(t *testing.T) {
	ctx := context.Background()

	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"block": { "header": {"height": "244591"}}}`)
	})
	chainParser, chainProxy, chainFetcher, closeServer, _, err := CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)
	block, err := chainFetcher.FetchLatestBlockNum(ctx)
	require.Greater(t, block, int64(0))
	require.NoError(t, err)

	chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	require.NoError(t, err)
	reqBlock, _ := chainMsg.RequestedBlock()
	require.Equal(t, int64(17), reqBlock)
	if closeServer != nil {
		closeServer()
	}
}

func TestParsingRequestedBlocksHeadersRest(t *testing.T) {
	ctx := context.Background()
	callbackHeaderNameToCheck := ""
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		header := r.Header.Get(callbackHeaderNameToCheck)
		if header != "" {
			fmt.Fprint(w, `{"block": { "header": {"height": "244590"}}}`)
		} else {
			fmt.Fprint(w, `{"block": { "header": {"height": "244591"}}}`)
		}
	})
	chainParser, chainRouter, _, closeServer, _, err := CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
	require.NoError(t, err)
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()
	parsingForCrafting, apiCollection, ok := chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	require.True(t, ok)
	collectionData := apiCollection.CollectionData
	headerParsingDirective, _, ok := chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_SET_LATEST_IN_METADATA)
	callbackHeaderNameToCheck = headerParsingDirective.GetApiName() // this causes the callback to modify the response to simulate a real behavior
	require.True(t, ok)
	block := 244590
	metadata := []pairingtypes.Metadata{{Name: headerParsingDirective.GetApiName(), Value: fmt.Sprintf(headerParsingDirective.FunctionTemplate, block)}}

	tests := []struct {
		desc           string
		metadata       []pairingtypes.Metadata
		block          int64
		requestedBlock int64
	}{
		{
			desc:           "no metadata",
			metadata:       []pairingtypes.Metadata{},
			block:          244591,
			requestedBlock: spectypes.LATEST_BLOCK,
		},
		{
			desc:           "with-metadata",
			metadata:       metadata,
			block:          244590,
			requestedBlock: 244590,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			chainMessage, err := chainParser.ParseMsg(parsingForCrafting.ApiName, []byte{}, collectionData.Type, test.metadata, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			require.NoError(t, err)
			latestReqBlock, _ := chainMessage.RequestedBlock()
			require.Equal(t, test.requestedBlock, latestReqBlock)
			reply, _, _, _, _, err := chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
			require.NoError(t, err)
			parserInput, err := FormatResponseForParsing(reply.RelayReply, chainMessage)
			require.NoError(t, err)
			parsedInput := parser.ParseBlockFromReply(parserInput, parsingForCrafting.ResultParsing, nil)
			require.Equal(t, test.block, parsedInput.GetBlock())
		})
	}
}

func TestSettingRequestedBlocksHeadersRest(t *testing.T) {
	ctx := context.Background()
	callbackHeaderNameToCheck := ""
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		header := r.Header.Get(callbackHeaderNameToCheck)
		if header != "" {
			parsedBlock, err := strconv.ParseUint(header, 0, 64)
			require.NoError(t, err)
			if parsedBlock < 244591 {
				fmt.Fprintf(w, `{"block": { "header": {"height": "%d"}}}`, parsedBlock)
				return
			}
		}
		fmt.Fprint(w, `{"block": { "header": {"height": "244591"}}}`)
	})
	chainParser, chainRouter, _, closeServer, _, err := CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
	require.NoError(t, err)
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()
	parsingForCrafting, apiCollection, ok := chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	require.True(t, ok)
	collectionData := apiCollection.CollectionData
	headerParsingDirective, _, ok := chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_SET_LATEST_IN_METADATA)
	callbackHeaderNameToCheck = headerParsingDirective.GetApiName() // this causes the callback to modify the response to simulate a real behavior
	require.True(t, ok)
	block := 244590
	metadata := []pairingtypes.Metadata{{Name: headerParsingDirective.GetApiName(), Value: fmt.Sprintf(headerParsingDirective.FunctionTemplate, block)}}

	tests := []struct {
		desc           string
		metadata       []pairingtypes.Metadata
		block          int64
		requestedBlock int64
	}{
		// Disabled due to inconsistency in cosmos sdk when adding these headers
		// {
		// 	desc:           "no metadata",
		// 	metadata:       []pairingtypes.Metadata{},
		// 	block:          244589,
		// 	requestedBlock: spectypes.LATEST_BLOCK,
		// },
		{
			desc:           "with-metadata",
			metadata:       metadata,
			block:          244590,
			requestedBlock: 244590,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			chainMessage, err := chainParser.ParseMsg(parsingForCrafting.ApiName, []byte{}, collectionData.Type, test.metadata, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			require.NoError(t, err)
			latestReqBlock, _ := chainMessage.RequestedBlock()
			require.Equal(t, test.requestedBlock, latestReqBlock)
			chainMessage.UpdateLatestBlockInMessage(test.block, true) // will update the block only if it's a latest request
			latestReqBlock, _ = chainMessage.RequestedBlock()
			require.Equal(t, test.block, latestReqBlock) // expected behavior is that it doesn't change the original requested block
			reply, _, _, _, _, err := chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
			require.NoError(t, err)
			parserInput, err := FormatResponseForParsing(reply.RelayReply, chainMessage)
			require.NoError(t, err)
			parsedInput := parser.ParseBlockFromReply(parserInput, parsingForCrafting.ResultParsing, nil)
			require.Equal(t, test.block, parsedInput.GetBlock())
		})
	}
}

func TestRegexParsing(t *testing.T) {
	chainParser, _, _, closeServer, _, err := CreateChainLibMocks(context.Background(), "LAV1", spectypes.APIInterfaceRest, nil, nil, "../../", nil)
	require.NoError(t, err)
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()
	for _, api := range []string{
		"/cosmos/staking/v1beta1/delegations/",
		"/lavanet/lava/pairing/provider/lava@1e9ma89h83azrfnqqy0u255zqxq0xluza6ydf9n/",
		"/lavanet/lava/pairing/provider/lava@1e9ma89h83azrfnqqy0u255zqxq0xluza6ydf9n/ETH1",
	} {
		_, err := chainParser.ParseMsg(api, nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
	}
	for _, api := range []string{
		"/cosmos/staking/v1beta1/delegations/lava@17ym998u666u8w2qgjd5m7w7ydjqmu3mlgl7ua2/",
	} {
		chainMessage, err := chainParser.ParseMsg(api, nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		if err == nil {
			require.Equal(t, "Default-"+api, chainMessage.GetApi().GetName())
		} else {
			assert.ErrorIs(t, err, common.APINotSupportedError)
		}
	}
}
