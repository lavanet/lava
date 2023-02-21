package chainlib

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	TendermintStatusQuery = "status"
)

type ChainFetcher struct {
	endpoint    *lavasession.RPCProviderEndpoint
	chainProxy  ChainProxy
	chainParser ChainParser
}

func (cf *ChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return *cf.endpoint
}

func (cf *ChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cf.chainParser.GetSpecApiByTag(spectypes.GET_BLOCKNUM)
	if !ok {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(spectypes.GET_BLOCKNUM+" tag function not found", nil, &map[string]string{"chainID": cf.endpoint.ChainID, "APIInterface": cf.endpoint.ApiInterface})
	}
	chainMessage := CraftChainMessage(serviceApi, cf.chainParser)
	reply, _, _, err := cf.chainProxy.SendNodeMsg(ctx, nil, chainMessage)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError(spectypes.GET_BLOCKNUM+" failed sending chainMessage", err, &map[string]string{"chainID": cf.endpoint.ChainID, "APIInterface": cf.endpoint.ApiInterface})
	}
	parserInput, err := cf.formatResponseForParsing(reply, chainMessage)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}
	blockNum, err := parser.ParseBlockFromReply(parserInput, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Failed To Parse FetchLatestBlockNum", err, &map[string]string{
			"nodeUrl":  strings.Join(cf.endpoint.NodeUrl, ","),
			"Method":   serviceApi.GetName(),
			"Response": string(reply.Data),
		})
	}
	return blockNum, nil
}

func (cf *ChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cf.chainParser.GetSpecApiByTag(spectypes.GET_BLOCK_BY_NUM)
	if !ok {
		return "", utils.LavaFormatError(spectypes.GET_BLOCK_BY_NUM+" tag function not found", nil, &map[string]string{"chainID": cf.endpoint.ChainID, "APIInterface": cf.endpoint.ApiInterface})
	}
	if serviceApi.GetParsing().FunctionTemplate == "" {
		return "", utils.LavaFormatError(spectypes.GET_BLOCK_BY_NUM+" missing function template", nil, &map[string]string{"chainID": cf.endpoint.ChainID, "APIInterface": cf.endpoint.ApiInterface})
	}
	path := serviceApi.Name
	data := []byte(fmt.Sprintf(serviceApi.GetParsing().FunctionTemplate, blockNum))
	chainMessage, err := cf.chainParser.ParseMsg(path, data, serviceApi.ApiInterfaces[0].Type)
	if err != nil {
		return "", utils.LavaFormatError(spectypes.GET_BLOCK_BY_NUM+" failed parseMsg on function template", err, &map[string]string{"chainID": cf.endpoint.ChainID, "APIInterface": cf.endpoint.ApiInterface})
	}
	reply, _, _, err := cf.chainProxy.SendNodeMsg(ctx, nil, chainMessage)
	if err != nil {
		return "", utils.LavaFormatError(spectypes.GET_BLOCK_BY_NUM+" failed sending chainMessage", err, &map[string]string{"chainID": cf.endpoint.ChainID, "APIInterface": cf.endpoint.ApiInterface})
	}
	parserInput, err := cf.formatResponseForParsing(reply, chainMessage)
	if err != nil {
		return "", err
	}

	blockData, err := parser.ParseMessageResponse(parserInput, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return "", err
	}

	// blockData is an interface array with the parsed result in index 0.
	// we know to expect a string result for a hash.
	ret, ok := blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)
	if !ok {
		return "", utils.LavaFormatError("Failed to Convert blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)", nil, &map[string]string{"blockData": fmt.Sprintf("%v", blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX])})
	}
	return ret, nil
}

func (cf *ChainFetcher) formatResponseForParsing(reply *types.RelayReply, chainMessage ChainMessageForSend) (parsable parser.RPCInput, err error) {
	var parserInput parser.RPCInput
	respData := reply.Data
	rpcMessage := chainMessage.GetRPCMessage()
	if customParsingMessage, ok := rpcMessage.(chainproxy.CustomParsingMessage); ok {
		parserInput, err = customParsingMessage.NewParsableRPCInput(respData)
		if err != nil {
			return nil, utils.LavaFormatError(spectypes.GET_BLOCK_BY_NUM+" failed creating NewParsableRPCInput from CustomParsingMessage", err, &map[string]string{"chainID": cf.endpoint.ChainID, "APIInterface": cf.endpoint.ApiInterface})
		}
	} else {
		parserInput = chainproxy.DefaultParsableRPCInput(respData)
	}
	return parserInput, nil
}

func NewChainFetcher(ctx context.Context, chainProxy ChainProxy, chainParser ChainParser, endpoint *lavasession.RPCProviderEndpoint) *ChainFetcher {
	cf := &ChainFetcher{chainProxy: chainProxy, chainParser: chainParser, endpoint: endpoint}
	return cf
}

type LavaChainFetcher struct {
	clientCtx client.Context
}

func (lcf *LavaChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return lavasession.RPCProviderEndpoint{NodeUrl: []string{lcf.clientCtx.NodeURI}, ChainID: "Lava-node", ApiInterface: "tendermintrpc"}
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

func NewLavaChainFetcher(ctx context.Context, clientCtx client.Context) *LavaChainFetcher {
	lcf := &LavaChainFetcher{clientCtx: clientCtx}
	return lcf
}
