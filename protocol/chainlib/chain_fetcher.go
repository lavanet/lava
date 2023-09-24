package chainlib

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/protocol/performance"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
		verifications, err := cf.chainParser.GetVerifications(addons)
		if err != nil {
			return err
		}
		if len(verifications) == 0 {
			utils.LavaFormatDebug("no verifications for NodeUrl", utils.Attribute{Key: "url", Value: url.String()})
		}
		latestBlock, err := cf.FetchLatestBlockNum(ctx)
		if err != nil {
			return err
		}
		for _, verification := range verifications {
			// we give several chances for starting up
			var err error
			for attempts := 0; attempts < 3; attempts++ {
				err = cf.Verify(ctx, verification, uint64(latestBlock))
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

func (cf *ChainFetcher) populateCache(relayData *pairingtypes.RelayPrivateData, reply *pairingtypes.RelayReply, requestedBlockHash []byte, finalized bool) {
	if requestedBlockHash != nil || finalized {
		new_ctx := context.Background()
		new_ctx, cancel := context.WithTimeout(new_ctx, common.DataReliabilityTimeoutIncrease)
		defer cancel()
		err := cf.cache.SetEntry(new_ctx, relayData, requestedBlockHash, cf.endpoint.ChainID, reply, finalized, "", nil)
		if err != nil && !performance.NotInitialisedError.Is(err) {
			utils.LavaFormatWarning("chain fetcher error updating cache with new entry", err)
		}
	}
}

func (cf *ChainFetcher) Verify(ctx context.Context, verification VerificationContainer, latestBlock uint64) error {
	parsing := &verification.ParseDirective
	collectionType := verification.ConnectionType
	path := parsing.ApiName
	data := []byte(fmt.Sprintf(parsing.FunctionTemplate))
	chainMessage, err := CraftChainMessage(parsing, collectionType, cf.chainParser, &CraftData{Path: path, Data: data, ConnectionType: collectionType}, cf.ChainFetcherMetadata())
	if err != nil {
		return utils.LavaFormatError("[-] verify failed creating chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}

	reply, _, _, err := cf.chainRouter.SendNodeMsg(ctx, nil, chainMessage, []string{verification.Extension})
	if err != nil {
		return utils.LavaFormatWarning("[-] verify failed sending chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}

	parserInput, err := FormatResponseForParsing(reply, chainMessage)
	if err != nil {
		return err
	}
	parsedResult, err := parser.ParseFromReply(parserInput, parsing.ResultParsing)
	if err != nil {
		return utils.LavaFormatWarning("[-] verify failed to parse result", err, []utils.Attribute{
			{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
			{Key: "Method", Value: parsing.GetApiName()},
			{Key: "Response", Value: string(reply.Data)},
		}...)
	}
	if verification.LatestDistance != 0 && latestBlock != 0 {
		parsedResultAsNumber, err := strconv.ParseUint(parsedResult, 0, 64)
		if err != nil {
			return utils.LavaFormatWarning("[-] verify failed to parse result as number", err, []utils.Attribute{
				{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
				{Key: "Method", Value: parsing.GetApiName()},
				{Key: "Response", Value: string(reply.Data)},
				{Key: "parsedResult", Value: parsedResult},
			}...)
		}
		if parsedResultAsNumber > latestBlock {
			return utils.LavaFormatWarning("[-] verify failed parsed result is greater than latestBlock", err, []utils.Attribute{
				{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
				{Key: "Method", Value: parsing.GetApiName()},
				{Key: "latestBlock", Value: latestBlock},
				{Key: "parsedResult", Value: parsedResultAsNumber},
			}...)
		}
		if latestBlock-parsedResultAsNumber < verification.LatestDistance {
			return utils.LavaFormatWarning("[-] verify failed expected block distance is not sufficient", err, []utils.Attribute{
				{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
				{Key: "Method", Value: parsing.GetApiName()},
				{Key: "latestBlock", Value: latestBlock},
				{Key: "parsedResult", Value: parsedResultAsNumber},
				{Key: "expected", Value: verification.LatestDistance},
			}...)
		}
	}
	// some verifications only want the response to be valid, and don't care about the value
	if verification.Value != "*" && verification.Value != "" {
		if parsedResult != verification.Value {
			return utils.LavaFormatWarning("[-] verify failed expected and received are different", err, []utils.Attribute{
				{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
				{Key: "Method", Value: parsing.GetApiName()},
				{Key: "Response", Value: string(reply.Data)},
			}...)
		}
	}
	utils.LavaFormatInfo("[+] verified successfully", utils.Attribute{Key: "endpoint", Value: cf.endpoint.String()}, utils.Attribute{Key: "verification", Value: verification.Name}, utils.Attribute{Key: "value", Value: parsedResult}, utils.Attribute{Key: "verificationKey", Value: verification.VerificationKey})
	return nil
}

func (cf *ChainFetcher) ChainFetcherMetadata() []pairingtypes.Metadata {
	ret := []pairingtypes.Metadata{
		{Name: ChainFetcherHeaderName, Value: cf.FetchEndpoint().NetworkAddress.Address},
	}
	return ret
}

func (cf *ChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	parsing, collectionData, ok := cf.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	tagName := spectypes.FUNCTION_TAG_GET_BLOCKNUM.String()
	if !ok {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(tagName+" tag function not found", nil, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	chainMessage, err := CraftChainMessage(parsing, collectionData.Type, cf.chainParser, nil, cf.ChainFetcherMetadata())
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(tagName+" failed creating chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	reply, _, _, err := cf.chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatWarning(tagName+" failed sending chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	parserInput, err := FormatResponseForParsing(reply, chainMessage)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatWarning(tagName+" Failed formatResponseForParsing", err, []utils.Attribute{
			{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
			{Key: "Method", Value: parsing.ApiName},
			{Key: "Response", Value: string(reply.Data)},
		}...)
	}
	blockNum, err := parser.ParseBlockFromReply(parserInput, parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatWarning(tagName+" Failed to parse Response", err, []utils.Attribute{
			{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
			{Key: "Method", Value: parsing.ApiName},
			{Key: "Response", Value: string(reply.Data)},
		}...)
	}
	atomic.StoreInt64(&cf.latestBlock, blockNum)
	return blockNum, nil
}

func (cf *ChainFetcher) constructRelayData(conectionType string, path string, data []byte, requestBlock int64, addon string, extensions []string) *pairingtypes.RelayPrivateData {
	relayData := &pairingtypes.RelayPrivateData{
		ConnectionType: conectionType,
		ApiUrl:         path,
		Data:           data,
		RequestBlock:   requestBlock,
		ApiInterface:   cf.endpoint.ApiInterface,
		Metadata:       nil,
		Addon:          addon,
		Extensions:     extensions,
	}
	return relayData
}

func (cf *ChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	parsing, collectionData, ok := cf.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM)
	tagName := spectypes.FUNCTION_TAG_GET_BLOCK_BY_NUM.String()
	if !ok {
		return "", utils.LavaFormatError(tagName+" tag function not found", nil, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	if parsing.FunctionTemplate == "" {
		return "", utils.LavaFormatError(tagName+" missing function template", nil, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	path := parsing.ApiName
	data := []byte(fmt.Sprintf(parsing.FunctionTemplate, blockNum))
	chainMessage, err := CraftChainMessage(parsing, collectionData.Type, cf.chainParser, &CraftData{Path: path, Data: data, ConnectionType: collectionData.Type}, cf.ChainFetcherMetadata())
	if err != nil {
		return "", utils.LavaFormatError(tagName+" failed CraftChainMessage on function template", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	reply, _, _, err := cf.chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
	if err != nil {
		return "", utils.LavaFormatWarning(tagName+" failed sending chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}
	parserInput, err := FormatResponseForParsing(reply, chainMessage)
	if err != nil {
		return "", utils.LavaFormatWarning(tagName+" Failed formatResponseForParsing", err, []utils.Attribute{
			{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
			{Key: "Method", Value: parsing.ApiName},
			{Key: "Response", Value: string(reply.Data)},
		}...)
	}

	res, err := parser.ParseMessageResponse(parserInput, parsing.ResultParsing)
	if err != nil {
		return "", utils.LavaFormatWarning(tagName+" Failed ParseMessageResponse", err, []utils.Attribute{
			{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
			{Key: "Method", Value: parsing.ApiName},
			{Key: "Response", Value: string(reply.Data)},
		}...)
	}
	_, _, blockDistanceToFinalization, _ := cf.chainParser.ChainBlockStats()
	latestBlock := atomic.LoadInt64(&cf.latestBlock) // assuming FetchLatestBlockNum is called before this one it's always true
	if latestBlock > 0 {
		finalized := spectypes.IsFinalizedBlock(blockNum, latestBlock, blockDistanceToFinalization)
		cf.populateCache(cf.constructRelayData(collectionData.Type, path, data, blockNum, "", nil), reply, []byte(res), finalized)
	}
	return res, nil
}

func NewChainFetcher(ctx context.Context, chainRouter ChainRouter, chainParser ChainParser, endpoint *lavasession.RPCProviderEndpoint, cache *performance.Cache) *ChainFetcher {
	cf := &ChainFetcher{chainRouter: chainRouter, chainParser: chainParser, endpoint: endpoint, cache: cache}
	return cf
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
		return nil, utils.LavaFormatWarning("result (reply.Data) is empty, can't be formatted for parsing", err)
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
	ChainFetcher
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
	cf := &DummyChainFetcher{ChainFetcher{chainRouter: chainRouter, chainParser: chainParser, endpoint: endpoint}}
	return cf
}
