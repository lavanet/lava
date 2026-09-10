package chainlib

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/parser"
	specutils "github.com/lavanet/lava/v5/utils/keeper"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	plantypes "github.com/lavanet/lava/v5/x/plans/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
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

	// fetch chain block stats
	allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData, blocksInFinalizationProof := apip.ChainBlockStats()

	// convert block time
	AverageBlockTime := time.Duration(apip.spec.AverageBlockTime) * time.Millisecond

	// check that the spec was set correctly
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

// TestCardanoSpec_HistoricalBlockRequestsAreNotLatest pins the block parsing of the
// Blockfrost-style `/blocks/{hash_or_number}` routes in the CARDANO spec.
//
// The routes used to declare `DEFAULT latest`, so `GET /blocks/1` was classified as a
// request for the latest block: the archive rule never fired, archive providers were
// not selected, and the archive CU multiplier was not applied. A numeric path parameter
// must now surface as the requested block, and a block hash must be captured as a
// requested hash (the consumer resolves hashes to heights through its cache) while the
// block itself falls back to latest.
func TestCardanoSpec_HistoricalBlockRequestsAreNotLatest(t *testing.T) {
	ctx := context.Background()
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"height": 12000000, "hash": "4ea1ba291e8eef538635a53e59fddba7810d1679631cc3aed7c8e6c4091a5b1f"}`)
	})
	chainParser, _, _, closeServer, _, err := CreateChainLibMocks(ctx, "CARDANO", spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
	require.NoError(t, err)
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()

	const blockHash = "4ea1ba291e8eef538635a53e59fddba7810d1679631cc3aed7c8e6c4091a5b1f"

	tests := []struct {
		path          string
		expectedBlock int64
		expectedHash  string
	}{
		{path: "/blocks/latest", expectedBlock: spectypes.LATEST_BLOCK},
		{path: "/blocks/1", expectedBlock: 1},
		{path: "/blocks/11500000", expectedBlock: 11500000},
		{path: "/blocks/1/next", expectedBlock: 1},
		{path: "/blocks/1/previous", expectedBlock: 1},
		{path: "/blocks/1/txs", expectedBlock: 1},
		{path: "/blocks/1/txs/cbor", expectedBlock: 1},
		{path: "/blocks/1/addresses", expectedBlock: 1},
		{path: "/blocks/" + blockHash, expectedBlock: spectypes.LATEST_BLOCK, expectedHash: blockHash},
		{path: "/blocks/" + blockHash + "/txs", expectedBlock: spectypes.LATEST_BLOCK, expectedHash: blockHash},
		// epoch and slot numbers are not block heights and must keep resolving to latest
		{path: "/epochs/1", expectedBlock: spectypes.LATEST_BLOCK},
		{path: "/blocks/slot/1", expectedBlock: spectypes.LATEST_BLOCK},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			chainMessage, err := chainParser.ParseMsg(test.path, nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			requestedBlock, _ := chainMessage.RequestedBlock()
			require.Equal(t, test.expectedBlock, requestedBlock)
			if test.expectedHash == "" {
				require.Empty(t, chainMessage.GetRequestedBlocksHashes())
			} else {
				require.Equal(t, []string{test.expectedHash}, chainMessage.GetRequestedBlocksHashes())
			}
		})
	}
}

// TestCardanoSpec_HistoricalBlockActivatesArchive checks the end-to-end effect of the
// parsing fix: with the archive extension enabled by policy, an old numeric block goes
// to archive with the archive CU multiplier, while latest and recent blocks do not.
func TestCardanoSpec_HistoricalBlockActivatesArchive(t *testing.T) {
	ctx := context.Background()
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"height": 12000000, "hash": "4ea1ba291e8eef538635a53e59fddba7810d1679631cc3aed7c8e6c4091a5b1f"}`)
	})
	chainParser, _, _, closeServer, _, err := CreateChainLibMocks(ctx, "CARDANO", spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
	require.NoError(t, err)
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()

	chainParser.SetPolicy(&plantypes.Policy{ChainPolicies: []plantypes.ChainPolicy{{ChainId: "CARDANO", Requirements: []plantypes.ChainRequirement{{Collection: spectypes.CollectionData{ApiInterface: spectypes.APIInterfaceRest}, Extensions: []string{"archive"}}}}}}, "CARDANO", spectypes.APIInterfaceRest)

	spec, err := specutils.GetASpec("CARDANO", "../../", nil, nil)
	require.NoError(t, err)
	var archiveRuleBlock uint64
	var archiveCuMultiplier uint64
	for _, apiCollection := range spec.ApiCollections {
		for _, extension := range apiCollection.Extensions {
			if extension.Name == "archive" {
				archiveRuleBlock = extension.Rule.Block
				archiveCuMultiplier = extension.CuMultiplier
			}
		}
	}
	require.NotZero(t, archiveRuleBlock)
	require.NotZero(t, archiveCuMultiplier)

	const latestBlock = uint64(12000000)
	baseCu := uint64(10) // /blocks/{hash_or_number} compute units

	tests := []struct {
		name       string
		path       string
		archive    bool
		expectedCu uint64
	}{
		{name: "latest", path: "/blocks/latest", archive: false, expectedCu: baseCu},
		{name: "recent block", path: fmt.Sprintf("/blocks/%d", latestBlock-1), archive: false, expectedCu: baseCu},
		{name: "block just inside the archive boundary", path: fmt.Sprintf("/blocks/%d", latestBlock-archiveRuleBlock-1), archive: true, expectedCu: baseCu * archiveCuMultiplier},
		{name: "genesis-era block", path: "/blocks/1", archive: true, expectedCu: baseCu * archiveCuMultiplier},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chainMessage, err := chainParser.ParseMsg(test.path, nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: latestBlock})
			require.NoError(t, err)
			if test.archive {
				require.Len(t, chainMessage.GetExtensions(), 1)
				require.Equal(t, "archive", chainMessage.GetExtensions()[0].Name)
			} else {
				require.Empty(t, chainMessage.GetExtensions())
			}
			require.Equal(t, test.expectedCu, chainMessage.GetApi().ComputeUnits)
		})
	}
}

// cardanoAddonPolicy builds a consumer policy for CARDANO that requires the given addons.
// Passing no addons reproduces every plan shipped today (empty or wildcard chain policies),
// which resolve to the base collection only.
func cardanoAddonPolicy(addons ...struct{ addOn, connectionType string }) *plantypes.Policy {
	requirements := []plantypes.ChainRequirement{}
	for _, addon := range addons {
		requirements = append(requirements, plantypes.ChainRequirement{
			Collection: spectypes.CollectionData{
				ApiInterface: spectypes.APIInterfaceRest,
				Type:         addon.connectionType,
				AddOn:        addon.addOn,
			},
			Mixed: true,
		})
	}
	return &plantypes.Policy{ChainPolicies: []plantypes.ChainPolicy{{ChainId: "CARDANO", Requirements: requirements}}}
}

// TestCardanoSpec_MempoolAndEvaluateAreAddons pins the resolution of lavanet/lava#2333:
// the three /mempool routes and the two /utils/txs/evaluate routes are not implemented by
// the self-hosted blockfrost-backend-ryo, so they must not be part of the mandatory base
// collection. They live in the "mempool" and "txs-evaluate" addons instead, which a
// provider opts into and a consumer policy must request.
func TestCardanoSpec_MempoolAndEvaluateAreAddons(t *testing.T) {
	ctx := context.Background()
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"height": 12000000}`)
	})
	chainParser, _, _, closeServer, _, err := CreateChainLibMocks(ctx, "CARDANO", spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
	require.NoError(t, err)
	defer func() {
		if closeServer != nil {
			closeServer()
		}
	}()

	mempoolAddon := struct{ addOn, connectionType string }{"mempool", http.MethodGet}
	evaluateAddon := struct{ addOn, connectionType string }{"txs-evaluate", http.MethodPost}

	addonRoutes := []struct {
		path           string
		connectionType string
	}{
		{path: "/mempool", connectionType: http.MethodGet},
		{path: "/mempool/4ea1ba291e8eef538635a53e59fddba7810d1679631cc3aed7c8e6c4091a5b1f", connectionType: http.MethodGet},
		{path: "/mempool/addresses/addr1qxqs59lphg8g6qndelq8xwqn60ag3aeyfcp33c2kdp46a09re5df3pzwwmyq946axfcejy5n4x0y99wqpgtp2gd0k09qsgy6pz", connectionType: http.MethodGet},
		{path: "/utils/txs/evaluate", connectionType: http.MethodPost},
		{path: "/utils/txs/evaluate/utxos", connectionType: http.MethodPost},
	}
	baseRoutes := []struct {
		path           string
		connectionType string
	}{
		{path: "/blocks/latest", connectionType: http.MethodGet},
		{path: "/tx/submit", connectionType: http.MethodPost},
	}

	// a policy without the addons - every plan in cookbook/plans today - reaches the base
	// collection only, and the five routes are rejected before they ever leave the consumer
	require.NoError(t, chainParser.SetPolicy(cardanoAddonPolicy(), "CARDANO", spectypes.APIInterfaceRest))
	for _, route := range addonRoutes {
		t.Run("rejected without addon "+route.path, func(t *testing.T) {
			_, err := ParseAndValidateMessage(chainParser, route.path, nil, route.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.Error(t, err)
		})
	}
	for _, route := range baseRoutes {
		t.Run("base reachable without addon "+route.path, func(t *testing.T) {
			_, err := ParseAndValidateMessage(chainParser, route.path, nil, route.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
		})
	}

	// once the policy asks for both addons the same routes resolve, and they carry the
	// addon so pairing can route them to a provider that declared it
	require.NoError(t, chainParser.SetPolicy(cardanoAddonPolicy(mempoolAddon, evaluateAddon), "CARDANO", spectypes.APIInterfaceRest))
	for _, route := range addonRoutes {
		t.Run("allowed with addon "+route.path, func(t *testing.T) {
			chainMessage, err := ParseAndValidateMessage(chainParser, route.path, nil, route.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			expectedAddon := mempoolAddon.addOn
			if route.connectionType == http.MethodPost {
				expectedAddon = evaluateAddon.addOn
			}
			require.Equal(t, expectedAddon, GetAddon(chainMessage))
		})
	}
	// the base collection is unaffected by the addons being present
	for _, route := range baseRoutes {
		t.Run("base unaffected by addon "+route.path, func(t *testing.T) {
			chainMessage, err := ParseAndValidateMessage(chainParser, route.path, nil, route.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			require.Equal(t, "", GetAddon(chainMessage))
		})
	}
}
