package chainlib

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/golang/protobuf/proto"
	formatter "github.com/lavanet/lava/v4/ecosystem/cache/format"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/parser"
	"github.com/lavanet/lava/v4/protocol/performance"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"golang.org/x/exp/slices"
)

const (
	TendermintStatusQuery  = "status"
	ChainFetcherHeaderName = "X-LAVA-Provider"
)

type ChainFetcherIf interface {
	FetchLatestBlockNum(ctx context.Context) (int64, error)
	FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error)
	FetchEndpoint() lavasession.RPCProviderEndpoint
	Validate(ctx context.Context) error
}

type ChainFetcher struct {
	endpoint    *lavasession.RPCProviderEndpoint
	chainRouter ChainRouter
	chainParser ChainParser
	cache       *performance.Cache
	latestBlock int64
}

func (cf *ChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return *cf.endpoint
}

func (cf *ChainFetcher) Validate(ctx context.Context) error {
	for _, url := range cf.endpoint.NodeUrls {
		addons := url.Addons
		verifications, err := cf.chainParser.GetVerifications(addons, url.InternalPath, cf.endpoint.ApiInterface)
		if err != nil {
			return err
		}
		if len(verifications) == 0 {
			utils.LavaFormatDebug("no verifications for NodeUrl", utils.Attribute{Key: "url", Value: url.String()})
		}
		var latestBlock int64
		for attempts := 0; attempts < 3; attempts++ {
			latestBlock, err = cf.FetchLatestBlockNum(ctx)
			if err == nil {
				break
			}
		}
		if err != nil {
			return err
		}
		for _, verification := range verifications {
			if slices.Contains(url.SkipVerifications, verification.Name) {
				utils.LavaFormatDebug("Skipping Verification", utils.LogAttr("verification", verification.Name))
				continue
			}
			// we give several chances for starting up
			var err error
			for attempts := 0; attempts < 3; attempts++ {
				err = cf.Verify(ctx, verification, uint64(latestBlock))
				if err == nil {
					break
				}
			}
			if err != nil {
				if verification.Severity == spectypes.ParseValue_Fail {
					return utils.LavaFormatError("invalid Verification on provider startup", err, utils.Attribute{Key: "Addons", Value: addons}, utils.Attribute{Key: "verification", Value: verification.Name})
				}
			}
		}
	}
	return nil
}

func (cf *ChainFetcher) populateCache(relayData *pairingtypes.RelayPrivateData, reply *pairingtypes.RelayReply, requestedBlockHash []byte, finalized bool) {
	if cf.cache.CacheActive() && (requestedBlockHash != nil || finalized) {
		new_ctx := context.Background()
		new_ctx, cancel := context.WithTimeout(new_ctx, common.DataReliabilityTimeoutIncrease)
		defer cancel()
		// provider side doesn't use SharedStateId, so we default it to empty so it wont have effect.

		hash, _, err := HashCacheRequest(relayData, cf.endpoint.ChainID)
		if err != nil {
			utils.LavaFormatError("populateCache Failed getting Hash for request", err)
			return
		}

		_, averageBlockTime, _, _ := cf.chainParser.ChainBlockStats()
		err = cf.cache.SetEntry(new_ctx, &pairingtypes.RelayCacheSet{
			RequestHash:      hash,
			BlockHash:        requestedBlockHash,
			ChainId:          cf.endpoint.ChainID,
			Response:         reply,
			Finalized:        finalized,
			OptionalMetadata: nil,
			RequestedBlock:   relayData.RequestBlock,
			SeenBlock:        relayData.SeenBlock, // seen block is latestBlock so it will hit consumers requesting it.
			SharedStateId:    "",
			AverageBlockTime: int64(averageBlockTime),
		})
		if err != nil {
			utils.LavaFormatWarning("chain fetcher error updating cache with new entry", err)
		}
	}
}

func getExtensionsForVerification(verification VerificationContainer, chainParser ChainParser) []string {
	extensions := []string{verification.Extension}

	collectionKey := CollectionKey{
		InternalPath:   verification.InternalPath,
		Addon:          verification.Addon,
		ConnectionType: verification.ConnectionType,
	}

	if chainParser.IsTagInCollection(spectypes.FUNCTION_TAG_SUBSCRIBE, collectionKey) {
		if verification.Extension == "" {
			extensions = []string{WebSocketExtension}
		} else {
			extensions = append(extensions, WebSocketExtension)
		}
	}

	return extensions
}

func (cf *ChainFetcher) Verify(ctx context.Context, verification VerificationContainer, latestBlock uint64) error {
	parsing := &verification.ParseDirective

	collectionType := verification.ConnectionType
	path := parsing.ApiName
	data := []byte(parsing.FunctionTemplate)

	if !verification.IsActive() {
		utils.LavaFormatDebug("skipping disabled verification", []utils.Attribute{
			{Key: "Extension", Value: verification.Extension},
			{Key: "Addon", Value: verification.Addon},
			utils.LogAttr("name", verification.Name),
			{Key: "chainID", Value: cf.endpoint.ChainID},
			{Key: "APIInterface", Value: cf.endpoint.ApiInterface},
		}...)
		return nil
	}

	// craft data for GET_BLOCK_BY_NUM verification that cannot use "earliest"
	// also check for %d because the data constructed assumes its presence
	if verification.ParseDirective.FunctionTag == spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM {
		if verification.LatestDistance != 0 && latestBlock != 0 {
			if latestBlock >= verification.LatestDistance {
				data = []byte(fmt.Sprintf(parsing.FunctionTemplate, latestBlock-verification.LatestDistance))
			} else {
				return utils.LavaFormatWarning("[-] verify failed getting non-earliest block for chainMessage", fmt.Errorf("latestBlock is smaller than latestDistance"),
					utils.LogAttr("path", path),
					utils.LogAttr("latest_block", latestBlock),
					utils.LogAttr("latest_distance", verification.LatestDistance),
				)
			}
		} else if verification.Value != "" {
			expectedValue, err := strconv.ParseInt(verification.Value, 10, 64)
			if err != nil {
				return utils.LavaFormatError("failed converting expected value to number", err, utils.LogAttr("value", verification.Value))
			}
			data = []byte(fmt.Sprintf(parsing.FunctionTemplate, expectedValue))
		} else {
			return utils.LavaFormatWarning("[-] verification misconfiguration", fmt.Errorf("FUNCTION_TAG_GET_BLOCK_BY_NUM defined without LatestDistance or LatestBlock or a proper expected value"),
				utils.LogAttr("latest_block", latestBlock),
				utils.LogAttr("latest_distance", verification.LatestDistance),
				utils.LogAttr("expected_value", verification.Value),
			)
		}
	}

	craftData := &CraftData{Path: path, Data: data, ConnectionType: collectionType, InternalPath: verification.InternalPath}
	chainMessage, err := CraftChainMessage(parsing, collectionType, cf.chainParser, craftData, cf.ChainFetcherMetadata())
	if err != nil {
		return utils.LavaFormatError("[-] verify failed creating chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}

	extensions := getExtensionsForVerification(verification, cf.chainParser)

	reply, _, _, proxyUrl, chainId, err := cf.chainRouter.SendNodeMsg(ctx, nil, chainMessage, extensions)
	if err != nil {
		return utils.LavaFormatWarning("[-] verify failed sending chainMessage", err,
			utils.LogAttr("chainID", cf.endpoint.ChainID),
			utils.LogAttr("APIInterface", cf.endpoint.ApiInterface),
			utils.LogAttr("extensions", extensions),
		)
	}
	if reply == nil || reply.RelayReply == nil {
		return utils.LavaFormatWarning("[-] verify failed sending chainMessage, reply or reply.RelayReply are nil", nil,
			utils.LogAttr("chainID", cf.endpoint.ChainID),
			utils.LogAttr("APIInterface", cf.endpoint.ApiInterface),
		)
	}

	parserInput, err := FormatResponseForParsing(reply.RelayReply, chainMessage)
	if err != nil {
		return utils.LavaFormatWarning("[-] verify failed to parse result", err,
			utils.LogAttr("chain_id", chainId),
			utils.LogAttr("Api_interface", cf.endpoint.ApiInterface),
		)
	}

	parsedInput := parser.ParseBlockFromReply(parserInput, parsing.ResultParsing, parsing.Parsers)
	if parsedInput.GetRawParsedData() == "" {
		return utils.LavaFormatWarning("[-] verify failed to parse result", nil,
			utils.LogAttr("chainId", chainId),
			utils.LogAttr("nodeUrl", proxyUrl.Url),
			utils.LogAttr("Method", parsing.GetApiName()),
			utils.LogAttr("Response", string(reply.RelayReply.Data)),
		)
	}

	parserError := parsedInput.GetParserError()
	if parserError != "" {
		return utils.LavaFormatWarning("[-] parser returned an error", nil,
			utils.LogAttr("error", parserError),
			utils.LogAttr("chainId", chainId),
			utils.LogAttr("nodeUrl", proxyUrl.Url),
			utils.LogAttr("Method", parsing.GetApiName()),
			utils.LogAttr("Response", string(reply.RelayReply.Data)),
		)
	}
	if verification.LatestDistance != 0 && latestBlock != 0 && verification.ParseDirective.FunctionTag != spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM {
		parsedResultAsNumber := parsedInput.GetBlock()
		if parsedResultAsNumber == spectypes.NOT_APPLICABLE {
			return utils.LavaFormatWarning("[-] verify failed to parse result as number", nil,
				utils.LogAttr("chainId", chainId),
				utils.LogAttr("nodeUrl", proxyUrl.Url),
				utils.LogAttr("Method", parsing.GetApiName()),
				utils.LogAttr("Response", string(reply.RelayReply.Data)),
				utils.LogAttr("rawParsedData", parsedInput.GetRawParsedData()),
			)
		}
		uint64ParsedResultAsNumber := uint64(parsedResultAsNumber)
		if uint64ParsedResultAsNumber > latestBlock {
			return utils.LavaFormatWarning("[-] verify failed parsed result is greater than latestBlock", nil,
				utils.LogAttr("chainId", chainId),
				utils.LogAttr("nodeUrl", proxyUrl.Url),
				utils.LogAttr("Method", parsing.GetApiName()),
				utils.LogAttr("latestBlock", latestBlock),
				utils.LogAttr("parsedResult", uint64ParsedResultAsNumber),
			)
		}
		if latestBlock-uint64ParsedResultAsNumber < verification.LatestDistance {
			return utils.LavaFormatWarning("[-] verify failed expected block distance is not sufficient", nil,
				utils.LogAttr("chainId", chainId),
				utils.LogAttr("nodeUrl", proxyUrl.Url),
				utils.LogAttr("Method", parsing.GetApiName()),
				utils.LogAttr("latestBlock", latestBlock),
				utils.LogAttr("parsedResult", uint64ParsedResultAsNumber),
				utils.LogAttr("expected", verification.LatestDistance),
			)
		}
	}
	// some verifications only want the response to be valid, and don't care about the value
	if verification.Value != "*" && verification.Value != "" && verification.ParseDirective.FunctionTag != spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM {
		rawData := parsedInput.GetRawParsedData()
		if rawData != verification.Value {
			return utils.LavaFormatWarning("[-] verify failed expected and received are different", nil,
				utils.LogAttr("chainId", chainId),
				utils.LogAttr("nodeUrl", proxyUrl.Url),
				utils.LogAttr("rawParsedBlock", rawData),
				utils.LogAttr("verification.Value", verification.Value),
				utils.LogAttr("Method", parsing.GetApiName()),
				utils.LogAttr("Extension", verification.Extension),
				utils.LogAttr("Addon", verification.Addon),
				utils.LogAttr("Verification", verification.Name),
			)
		}
	}

	utils.LavaFormatInfo("[+] verified successfully",
		utils.LogAttr("chainId", chainId),
		utils.LogAttr("nodeUrl", proxyUrl.Url),
		utils.LogAttr("verification", verification.Name),
		utils.LogAttr("block", parsedInput.GetBlock()),
		utils.LogAttr("rawData", parsedInput.GetRawParsedData()),
		utils.LogAttr("verificationKey", verification.VerificationKey),
		utils.LogAttr("apiInterface", cf.endpoint.ApiInterface),
		utils.LogAttr("internalPath", proxyUrl.InternalPath),
	)
	return nil
}

func (cf *ChainFetcher) ChainFetcherMetadata() []pairingtypes.Metadata {
	ret := []pairingtypes.Metadata{
		{Name: ChainFetcherHeaderName, Value: cf.FetchEndpoint().NetworkAddress.Address},
	}
	return ret
}

func (cf *ChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	parsing, apiCollection, ok := cf.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	tagName := spectypes.FUNCTION_TAG_GET_BLOCKNUM.String()
	if !ok {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(tagName+" tag function not found", nil, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	collectionData := apiCollection.CollectionData
	var craftData *CraftData
	if parsing.FunctionTemplate != "" {
		path := parsing.ApiName
		data := []byte(parsing.FunctionTemplate)
		craftData = &CraftData{Path: path, Data: data, ConnectionType: collectionData.Type}
	}
	chainMessage, err := CraftChainMessage(parsing, collectionData.Type, cf.chainParser, craftData, cf.ChainFetcherMetadata())
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(tagName+" failed creating chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	reply, _, _, proxyUrl, chainId, err := cf.chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatDebug(tagName+" failed sending chainMessage", []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}, {Key: "error", Value: err}}...)
	}
	parserInput, err := FormatResponseForParsing(reply.RelayReply, chainMessage)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatDebug(tagName+" Failed formatResponseForParsing", []utils.Attribute{
			{Key: "chainId", Value: chainId},
			{Key: "nodeUrl", Value: proxyUrl.Url},
			{Key: "Method", Value: parsing.ApiName},
			{Key: "Response", Value: string(reply.RelayReply.Data)},
			{Key: "error", Value: err},
		}...)
	}
	parsedInput := parser.ParseBlockFromReply(parserInput, parsing.ResultParsing, parsing.Parsers)
	blockNum := parsedInput.GetBlock()
	if blockNum == spectypes.NOT_APPLICABLE {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatDebug(tagName+" Failed to parse Response", []utils.Attribute{
			{Key: "chainId", Value: chainId},
			{Key: "nodeUrl", Value: proxyUrl.Url},
			{Key: "Method", Value: parsing.ApiName},
			{Key: "Response", Value: string(reply.RelayReply.Data)},
			{Key: "error", Value: err},
		}...)
	}
	atomic.StoreInt64(&cf.latestBlock, blockNum)
	return blockNum, nil
}

func (cf *ChainFetcher) constructRelayData(conectionType string, path string, data []byte, requestBlock int64, addon string, extensions []string, latestBlock int64) *pairingtypes.RelayPrivateData {
	relayData := &pairingtypes.RelayPrivateData{
		ConnectionType: conectionType,
		ApiUrl:         path,
		Data:           data,
		RequestBlock:   requestBlock,
		ApiInterface:   cf.endpoint.ApiInterface,
		Metadata:       nil,
		Addon:          addon,
		Extensions:     extensions,
		SeenBlock:      latestBlock,
	}
	return relayData
}

func (cf *ChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	parsing, apiCollection, ok := cf.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM)
	tagName := spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM.String()
	if !ok {
		return "", utils.LavaFormatError(tagName+" tag function not found", nil, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	collectionData := apiCollection.CollectionData

	if parsing.FunctionTemplate == "" {
		return "", utils.LavaFormatError(tagName+" missing function template", nil, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	path := parsing.ApiName
	data := []byte(fmt.Sprintf(parsing.FunctionTemplate, blockNum))
	chainMessage, err := CraftChainMessage(parsing, collectionData.Type, cf.chainParser, &CraftData{Path: path, Data: data, ConnectionType: collectionData.Type}, cf.ChainFetcherMetadata())
	if err != nil {
		return "", utils.LavaFormatError(tagName+" failed CraftChainMessage on function template", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	start := time.Now()
	reply, _, _, proxyUrl, chainId, err := cf.chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
	if err != nil {
		timeTaken := time.Since(start)
		return "", utils.LavaFormatDebug(tagName+" failed sending chainMessage", []utils.Attribute{{Key: "sendTime", Value: timeTaken}, {Key: "error", Value: err}, {Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	parserInput, err := FormatResponseForParsing(reply.RelayReply, chainMessage)
	if err != nil {
		return "", utils.LavaFormatDebug(tagName+" Failed formatResponseForParsing", []utils.Attribute{
			{Key: "error", Value: err},
			{Key: "chainId", Value: chainId},
			{Key: "nodeUrl", Value: proxyUrl.Url},
			{Key: "Method", Value: parsing.ApiName},
			{Key: "Response", Value: string(reply.RelayReply.Data)},
		}...)
	}

	res, err := parser.ParseBlockHashFromReplyAndDecode(parserInput, parsing.ResultParsing, parsing.Parsers)
	if err != nil {
		return "", utils.LavaFormatDebug(tagName+" Failed ParseMessageResponse", []utils.Attribute{
			{Key: "error", Value: err},
			{Key: "chainId", Value: chainId},
			{Key: "nodeUrl", Value: proxyUrl.Url},
			{Key: "Method", Value: parsing.ApiName},
			{Key: "Response", Value: string(reply.RelayReply.Data)},
		}...)
	}
	_, _, blockDistanceToFinalization, _ := cf.chainParser.ChainBlockStats()
	latestBlock := atomic.LoadInt64(&cf.latestBlock) // assuming FetchLatestBlockNum is called before this one it's always true
	if latestBlock > 0 {
		finalized := spectypes.IsFinalizedBlock(blockNum, latestBlock, int64(blockDistanceToFinalization))
		isNodeError, _ := chainMessage.CheckResponseError(reply.RelayReply.Data, reply.StatusCode)
		if !isNodeError { // skip cache populate on node errors, this is a protection but should never get here with node error as we parse the result prior.
			cf.populateCache(cf.constructRelayData(collectionData.Type, path, data, blockNum, "", nil, latestBlock), reply.RelayReply, []byte(res), finalized)
		}
	}
	return res, nil
}

type ChainFetcherOptions struct {
	ChainRouter ChainRouter
	ChainParser ChainParser
	Endpoint    *lavasession.RPCProviderEndpoint
	Cache       *performance.Cache
}

func NewChainFetcher(ctx context.Context, options *ChainFetcherOptions) *ChainFetcher {
	return &ChainFetcher{
		chainRouter: options.ChainRouter,
		chainParser: options.ChainParser,
		endpoint:    options.Endpoint,
		cache:       options.Cache,
	}
}

type LavaChainFetcher struct {
	clientCtx client.Context
}

func (lcf *LavaChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return lavasession.RPCProviderEndpoint{NodeUrls: []common.NodeUrl{{Url: lcf.clientCtx.NodeURI}}, ChainID: "Lava-node", ApiInterface: "tendermintrpc"}
}

func (lcf *LavaChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	resultStatus, err := lcf.clientCtx.Client.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resultStatus.SyncInfo.LatestBlockHeight, nil
}

func (lcf *LavaChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	resultStatus, err := lcf.clientCtx.Client.Status(ctx)
	if err != nil {
		return "", err
	}
	return resultStatus.SyncInfo.LatestBlockHash.String(), nil
}

func (lcf *LavaChainFetcher) FetchChainID(ctx context.Context) (string, string, error) {
	return "", "", utils.LavaFormatError("FetchChainID not supported for lava chain fetcher", nil)
}

func NewLavaChainFetcher(ctx context.Context, clientCtx client.Context) *LavaChainFetcher {
	lcf := &LavaChainFetcher{clientCtx: clientCtx}
	return lcf
}

func FormatResponseForParsing(reply *pairingtypes.RelayReply, chainMessage ChainMessageForSend) (parsable parser.RPCInput, err error) {
	var parserInput parser.RPCInput
	respData := reply.Data
	if len(respData) == 0 {
		return nil, utils.LavaFormatDebug("result (reply.Data) is empty, can't be formatted for parsing", utils.Attribute{Key: "error", Value: err})
	}
	rpcMessage := chainMessage.GetRPCMessage()
	if customParsingMessage, ok := rpcMessage.(chainproxy.CustomParsingMessage); ok {
		parserInput, err = customParsingMessage.NewParsableRPCInput(respData)
		if err != nil {
			return nil, utils.LavaFormatError("failed creating NewParsableRPCInput from CustomParsingMessage", err, utils.Attribute{Key: "data", Value: string(respData)})
		}
	} else {
		parserInput = chainproxy.DefaultParsableRPCInput(respData)
	}
	return parserInput, nil
}

type DummyChainFetcher struct {
	*ChainFetcher
}

func (cf *DummyChainFetcher) Validate(ctx context.Context) error {
	for _, url := range cf.endpoint.NodeUrls {
		addons := url.Addons
		verifications, err := cf.chainParser.GetVerifications(addons, url.InternalPath, cf.endpoint.ApiInterface)
		if err != nil {
			return err
		}
		if len(verifications) == 0 {
			utils.LavaFormatDebug("no verifications for NodeUrl", utils.Attribute{Key: "url", Value: url.String()})
		}
		for _, verification := range verifications {
			// we give several chances for starting up
			var err error
			for attempts := 0; attempts < 3; attempts++ {
				err = cf.Verify(ctx, verification, 0)
				if err == nil {
					break
				}
			}
			if err != nil {
				return utils.LavaFormatError("invalid Verification on provider startup", err, utils.Attribute{Key: "Addons", Value: addons}, utils.Attribute{Key: "verification", Value: verification.Name})
			}
		}
	}
	return nil
}

// overwrite this
func (cf *DummyChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	return 0, nil
}

// overwrite this too
func (cf *DummyChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	return "dummy", nil
}

func NewVerificationsOnlyChainFetcher(ctx context.Context, chainRouter ChainRouter, chainParser ChainParser, endpoint *lavasession.RPCProviderEndpoint) *DummyChainFetcher {
	cfi := ChainFetcher{chainRouter: chainRouter, chainParser: chainParser, endpoint: endpoint}
	cf := &DummyChainFetcher{ChainFetcher: &cfi}
	return cf
}

// this method will calculate the request hash by changing the original object, and returning the data back to it after calculating the hash
// couldn't be used in parallel
func HashCacheRequest(relayData *pairingtypes.RelayPrivateData, chainId string) ([]byte, func([]byte) []byte, error) {
	originalData := relayData.Data
	originalSalt := relayData.Salt
	originalRequestedBlock := relayData.RequestBlock
	originalSeenBlock := relayData.SeenBlock
	defer func() {
		// return all information back to the object on defer (in any case)
		relayData.Data = originalData
		relayData.Salt = originalSalt
		relayData.RequestBlock = originalRequestedBlock
		relayData.SeenBlock = originalSeenBlock
	}()

	// we need to remove some data from the request so the cache will hit properly.
	inputFormatter, outputFormatter := formatter.FormatterForRelayRequestAndResponse(relayData.ApiInterface)
	relayData.Data = inputFormatter(relayData.Data) // remove id from request.
	relayData.Salt = nil                            // remove salt
	relayData.SeenBlock = 0                         // remove seen block
	// we remove the discrepancy of requested block from the hash, and add it on the cache side instead
	// this is due to the fact that we don't know the latest seen block at this moment, as on shared state
	// only the cache has this information. we make sure the hashing at this stage does not include the requested block.
	// It does include it on the cache key side.
	relayData.RequestBlock = 0

	cashHash := &pairingtypes.CacheHash{
		Request: relayData,
		ChainId: chainId,
	}
	cashHashBytes, err := proto.Marshal(cashHash)
	if err != nil {
		return nil, outputFormatter, utils.LavaFormatError("Failed marshalling cash hash in HashCacheRequest", err)
	}

	// return the value
	return sigs.HashMsg(cashHashBytes), outputFormatter, nil
}
