package chainlib

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/parser"
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
}

func (cf *ChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return *cf.endpoint
}

func (cf *ChainFetcher) Validate(ctx context.Context) error {
	realChainID, specChainId, err := cf.FetchChainID(ctx)
	// Validate spec chain id
	if specChainId != realChainID {
		return utils.LavaFormatError("panic severity critical error, aborting support for chain api due to invalid chain ID, continuing with other endpoints", err, utils.Attribute{Key: "Spec chain ID", Value: specChainId}, utils.Attribute{Key: "Real chain ID", Value: realChainID}, utils.Attribute{Key: "endpoint", Value: cf.FetchEndpoint()})
	}
	return nil
}

func (cf *ChainFetcher) ChainFetcherMetadata() []pairingtypes.Metadata {
	ret := []pairingtypes.Metadata{
		{Name: ChainFetcherHeaderName, Value: cf.FetchEndpoint().NetworkAddress.Address},
	}
	return ret
}

func (cf *ChainFetcher) FetchChainID(ctx context.Context) (string, string, error) {
	parsing, collectionData, ok := cf.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_CHAIN_ID)
	tagName := spectypes.FUNCTION_TAG_GET_CHAIN_ID.String()
	// If parsing tag or wanted result does not exist
	// skip chain id check with warning
	if !ok || parsing.ResultParsing.DefaultValue == "" {
		utils.LavaFormatWarning("skipping chain ID validation, chain ID missing from the spec", nil,
			utils.Attribute{Key: "Chain ID", Value: cf.endpoint.ChainID},
		)
		return "", "", nil
	}

	chainMessage, err := CraftChainMessage(parsing, collectionData.Type, cf.chainParser, nil, cf.ChainFetcherMetadata())
	if err != nil {
		return "", "", utils.LavaFormatError(tagName+" failed creating chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}

	reply, _, _, err := cf.chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
	if err != nil {
		return "", "", utils.LavaFormatWarning(tagName+" failed sending chainMessage", err, []utils.Attribute{{Key: "chainID", Value: cf.endpoint.ChainID}, {Key: "APIInterface", Value: cf.endpoint.ApiInterface}}...)
	}

	parserInput, err := FormatResponseForParsing(reply, chainMessage)
	if err != nil {
		return "", "", err
	}
	specID, err := parser.ParseFromReply(parserInput, parsing.ResultParsing)
	if err != nil {
		return "", "", utils.LavaFormatWarning("Failed To Parse FetchChainID", err, []utils.Attribute{
			{Key: "nodeUrl", Value: cf.endpoint.UrlsString()},
			{Key: "Method", Value: parsing.GetApiName()},
			{Key: "Response", Value: string(reply.Data)},
		}...)
	}

	return specID, parsing.ResultParsing.DefaultValue, nil
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
	return blockNum, nil
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
	return res, nil
}

func NewChainFetcher(ctx context.Context, chainRouter ChainRouter, chainParser ChainParser, endpoint *lavasession.RPCProviderEndpoint) *ChainFetcher {
	cf := &ChainFetcher{chainRouter: chainRouter, chainParser: chainParser, endpoint: endpoint}
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
			return nil, utils.LavaFormatError("failed creating NewParsableRPCInput from CustomParsingMessage", err)
		}
	} else {
		parserInput = chainproxy.DefaultParsableRPCInput(respData)
	}
	return parserInput, nil
}

type DummyChainFetcher struct {
	endpoint *lavasession.RPCProviderEndpoint
}

func (cf *DummyChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return *cf.endpoint
}

func (cf *DummyChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	return 0, nil
}

func (cf *DummyChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	return "dummy", nil
}

func (cf *DummyChainFetcher) Validate(ctx context.Context) error {
	return nil
}

func NewDummyChainFetcher(ctx context.Context, endpoint *lavasession.RPCProviderEndpoint) *DummyChainFetcher {
	cf := &DummyChainFetcher{endpoint: endpoint}
	return cf
}
