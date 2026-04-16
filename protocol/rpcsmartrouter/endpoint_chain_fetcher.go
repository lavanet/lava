package rpcsmartrouter

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/parser"
	pairingtypes "github.com/lavanet/lava/v5/types/relay"
	spectypes "github.com/lavanet/lava/v5/types/spec"
	"github.com/lavanet/lava/v5/utils"
)

// EndpointChainFetcher implements chaintracker.ChainFetcher for direct RPC endpoints.
// It enables per-endpoint ChainTracker to continuously poll block data.
type EndpointChainFetcher struct {
	endpoint         *lavasession.Endpoint
	directConnection lavasession.DirectRPCConnection
	chainParser      chainlib.ChainParser
	chainID          string
	apiInterface     string
	latestBlock      int64

	// Metadata for requests
	endpointURL string
}

// NewEndpointChainFetcher creates a new ChainFetcher for a direct RPC endpoint.
func NewEndpointChainFetcher(
	endpoint *lavasession.Endpoint,
	directConnection lavasession.DirectRPCConnection,
	chainParser chainlib.ChainParser,
	chainID string,
	apiInterface string,
) *EndpointChainFetcher {
	return &EndpointChainFetcher{
		endpoint:         endpoint,
		directConnection: directConnection,
		chainParser:      chainParser,
		chainID:          chainID,
		apiInterface:     apiInterface,
		endpointURL:      endpoint.NetworkAddress,
	}
}

// FetchLatestBlockNum fetches the latest block number from the endpoint.
// Uses spec-driven parsing to support any chain type (EVM, Tendermint, REST, etc.).
func (ecf *EndpointChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	parsing, apiCollection, ok := ecf.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	tagName := spectypes.FUNCTION_TAG_GET_BLOCKNUM.String()
	if !ok {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(tagName+" tag function not found", nil,
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("apiInterface", ecf.apiInterface),
		)
	}

	collectionData := apiCollection.CollectionData

	// Get the request data from the function template
	if parsing.FunctionTemplate == "" {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(tagName+" missing function template", nil,
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("apiInterface", ecf.apiInterface),
		)
	}

	requestData := []byte(parsing.FunctionTemplate)

	// Send request via direct RPC connection
	responseData, err := ecf.sendRawRequest(ctx, requestData, collectionData.Type, parsing.ApiName)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatDebug(tagName+" failed sending request",
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("apiInterface", ecf.apiInterface),
			utils.LogAttr("endpoint", ecf.endpointURL),
			utils.LogAttr("error", err),
		)
	}

	// Craft chain message for response parsing (needed for FormatResponseForParsing)
	craftData := &chainlib.CraftData{
		Path:           parsing.ApiName,
		Data:           requestData,
		ConnectionType: collectionData.Type,
	}
	chainMessage, err := chainlib.CraftChainMessage(parsing, collectionData.Type, ecf.chainParser, craftData, ecf.chainFetcherMetadata())
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(tagName+" failed creating chainMessage for parsing", err,
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("apiInterface", ecf.apiInterface),
		)
	}

	// Parse the response using spec-driven rules
	parserInput, err := chainlib.FormatResponseForParsing(&pairingtypes.RelayReply{Data: responseData}, chainMessage)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatDebug(tagName+" failed formatResponseForParsing",
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("endpoint", ecf.endpointURL),
			utils.LogAttr("method", parsing.ApiName),
			utils.LogAttr("response", parser.CapStringLen(string(responseData))),
			utils.LogAttr("error", err),
		)
	}

	parsedInput := parser.ParseBlockFromReply(parserInput, parsing.ResultParsing, parsing.Parsers)
	blockNum := parsedInput.GetBlock()
	if blockNum == spectypes.NOT_APPLICABLE {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatDebug(tagName+" failed to parse response",
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("endpoint", ecf.endpointURL),
			utils.LogAttr("method", parsing.ApiName),
			utils.LogAttr("response", parser.CapStringLen(string(responseData))),
		)
	}

	atomic.StoreInt64(&ecf.latestBlock, blockNum)
	return blockNum, nil
}

// FetchBlockHashByNum fetches the block hash for a given block number.
// Used by ChainTracker for fork detection.
//
// For Solana-family chains, if the endpoint returns error code -32004
// ("Block not available for slot X"), this method retries with previous slot
// numbers (blockNum-1, blockNum-2, ...) up to maxBlockNotAvailableRetries times.
// This handles both propagation delays (the latest slot data hasn't reached the
// node yet) and skipped slots (Solana occasionally produces no block for a slot).
func (ecf *EndpointChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	parsing, apiCollection, ok := ecf.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM)
	tagName := spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM.String()
	if !ok {
		return "", utils.LavaFormatError(tagName+" tag function not found", nil,
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("apiInterface", ecf.apiInterface),
		)
	}

	collectionData := apiCollection.CollectionData

	if parsing.FunctionTemplate == "" {
		return "", utils.LavaFormatError(tagName+" missing function template", nil,
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("apiInterface", ecf.apiInterface),
		)
	}

	if blockNum < 0 {
		return "", utils.LavaFormatError(tagName+" invalid negative block number", nil,
			utils.LogAttr("blockNum", blockNum),
			utils.LogAttr("chainID", ecf.chainID),
		)
	}

	if !common.IsSolanaFamily(ecf.chainID) {
		hash, _, err := ecf.fetchSingleBlockHash(ctx, blockNum, parsing, collectionData.Type, tagName)
		return hash, err
	}

	fetchFn := func(fCtx context.Context, block int64) (string, []byte, error) {
		return ecf.fetchSingleBlockHash(fCtx, block, parsing, collectionData.Type, tagName)
	}
	hash, fetchedBlock, err := chainlib.FetchBlockHashWithSolanaRetry(ctx, blockNum, chainlib.SameSlotRetryDelay, fetchFn)
	if err != nil {
		return "", utils.LavaFormatError(tagName+" all block-not-available retries exhausted", err,
			utils.LogAttr("originalBlock", blockNum),
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("endpoint", ecf.endpointURL),
		)
	}
	if fetchedBlock != blockNum {
		utils.LavaFormatWarning("Chain Tracker fetched previous slot after block-not-available",
			nil,
			utils.LogAttr("originalBlock", blockNum),
			utils.LogAttr("fetchedBlock", fetchedBlock),
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("endpoint", ecf.endpointURL),
		)
	}
	return hash, nil
}

// fetchSingleBlockHash fetches the block hash for a single block number.
// Returns the hash, the raw response data (for error inspection), and any error.
func (ecf *EndpointChainFetcher) fetchSingleBlockHash(
	ctx context.Context,
	blockNum int64,
	parsing *spectypes.ParseDirective,
	connectionType string,
	tagName string,
) (string, []byte, error) {
	requestData := []byte(fmt.Sprintf(parsing.FunctionTemplate, blockNum))

	start := time.Now()
	responseData, err := ecf.sendRawRequest(ctx, requestData, connectionType, parsing.ApiName)
	if err != nil {
		timeTaken := time.Since(start)
		return "", nil, utils.LavaFormatDebug(tagName+" failed sending request",
			utils.LogAttr("sendTime", timeTaken),
			utils.LogAttr("error", err),
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("endpoint", ecf.endpointURL),
		)
	}

	craftData := &chainlib.CraftData{
		Path:           parsing.ApiName,
		Data:           requestData,
		ConnectionType: connectionType,
	}
	chainMessage, err := chainlib.CraftChainMessage(parsing, connectionType, ecf.chainParser, craftData, ecf.chainFetcherMetadata())
	if err != nil {
		return "", responseData, utils.LavaFormatError(tagName+" failed CraftChainMessage", err,
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("apiInterface", ecf.apiInterface),
		)
	}

	parserInput, err := chainlib.FormatResponseForParsing(&pairingtypes.RelayReply{Data: responseData}, chainMessage)
	if err != nil {
		return "", responseData, utils.LavaFormatDebug(tagName+" failed formatResponseForParsing",
			utils.LogAttr("error", err),
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("endpoint", ecf.endpointURL),
			utils.LogAttr("method", parsing.ApiName),
			utils.LogAttr("response", parser.CapStringLen(string(responseData))),
		)
	}

	res, err := parser.ParseBlockHashFromReplyAndDecode(parserInput, parsing.ResultParsing, parsing.Parsers)
	if err != nil {
		return "", responseData, utils.LavaFormatDebug(tagName+" failed ParseBlockHashFromReplyAndDecode",
			utils.LogAttr("error", err),
			utils.LogAttr("chainID", ecf.chainID),
			utils.LogAttr("endpoint", ecf.endpointURL),
			utils.LogAttr("method", parsing.ApiName),
			utils.LogAttr("response", parser.CapStringLen(string(responseData))),
		)
	}

	return res, responseData, nil
}

// FetchEndpoint returns the endpoint information for this fetcher.
// Required by chaintracker.ChainFetcher interface.
func (ecf *EndpointChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return lavasession.RPCProviderEndpoint{
		ChainID:      ecf.chainID,
		ApiInterface: ecf.apiInterface,
		NodeUrls:     []common.NodeUrl{{Url: ecf.endpointURL}},
	}
}

// CustomMessage sends a custom JSON-RPC / REST message to the endpoint.
// Used by SVMChainTracker to call getLatestBlockhash (which returns slot + block
// hash + block height together — a single call that has no equivalent in the
// generic FetchLatestBlockNum path). Returning an error here disables the per-
// endpoint ChainTracker on Solana, which in turn starves every per-endpoint
// metric that depends on OnNewBlock (latest_block, fetch_latest_success, …).
//
// The `path` argument is accepted for interface compatibility with
// chainlib.ChainFetcher.CustomMessage but is not needed here: POST callers
// (like SVMChainTracker) pass the body in `data` with `path=""`, and GET
// callers already encode the URL suffix in `data` per sendRawRequest's REST
// convention (see connectionType == "GET" branch below).
func (ecf *EndpointChainFetcher) CustomMessage(ctx context.Context, path string, data []byte, connectionType string, apiName string) ([]byte, error) {
	return ecf.sendRawRequest(ctx, data, connectionType, apiName)
}

// sendRawRequest sends a raw request to the endpoint and returns the response.
// For REST/GET requests, requestData is a URL path that must be appended to the base URL.
// For JSON-RPC/POST requests, requestData is the JSON body.
func (ecf *EndpointChainFetcher) sendRawRequest(ctx context.Context, requestData []byte, connectionType string, apiName string) ([]byte, error) {
	if ecf.directConnection == nil || !ecf.directConnection.IsHealthy() {
		return nil, fmt.Errorf("direct connection is not healthy for endpoint %s", ecf.endpointURL)
	}

	// REST GET: requestData is a URL path (e.g. "/cosmos/base/tendermint/v1beta1/blocks/latest")
	// Must be appended to the base URL and sent as an HTTP GET.
	if connectionType == "GET" {
		httpDoer, ok := ecf.directConnection.(lavasession.HTTPDirectRPCDoer)
		if !ok {
			return nil, fmt.Errorf("connection does not support HTTP requests for endpoint %s", ecf.endpointURL)
		}

		fullURL, err := joinURLPath(ecf.directConnection.GetURL(), string(requestData))
		if err != nil {
			return nil, fmt.Errorf("failed to build REST URL: %w", err)
		}

		resp, err := httpDoer.DoHTTPRequest(ctx, lavasession.HTTPRequestParams{
			Method: "GET",
			URL:    fullURL,
		})
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			return nil, &lavasession.HTTPStatusError{
				StatusCode: resp.StatusCode,
				Status:     fmt.Sprintf("%d", resp.StatusCode),
				Body:       resp.Body,
			}
		}
		return resp.Body, nil
	}

	// JSON-RPC / Tendermint RPC / POST: send requestData as body
	headers := map[string]string{"Content-Type": "application/json"}
	response, err := ecf.directConnection.SendRequest(ctx, requestData, headers)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

// chainFetcherMetadata returns metadata for constructing chain messages.
func (ecf *EndpointChainFetcher) chainFetcherMetadata() []pairingtypes.Metadata {
	return nil // No special metadata needed for block tracking
}

// GetLatestBlock returns the last known latest block number.
func (ecf *EndpointChainFetcher) GetLatestBlock() int64 {
	return atomic.LoadInt64(&ecf.latestBlock)
}
