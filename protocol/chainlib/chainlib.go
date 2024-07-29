package chainlib

import (
	"context"
	"fmt"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

var (
	IgnoreSubscriptionNotConfiguredError     = true
	IgnoreSubscriptionNotConfiguredErrorFlag = "ignore-subscription-not-configured-error"
)

func NewChainParser(apiInterface string) (chainParser ChainParser, err error) {
	switch apiInterface {
	case spectypes.APIInterfaceJsonRPC:
		return NewJrpcChainParser()
	case spectypes.APIInterfaceTendermintRPC:
		return NewTendermintRpcChainParser()
	case spectypes.APIInterfaceRest:
		return NewRestChainParser()
	case spectypes.APIInterfaceGrpc:
		return NewGrpcChainParser()
	}
	return nil, fmt.Errorf("chainParser for apiInterface (%s) not found", apiInterface)
}

func NewChainListener(
	ctx context.Context,
	listenEndpoint *lavasession.RPCEndpoint,
	relaySender RelaySender,
	healthReporter HealthReporter,
	rpcConsumerLogs *metrics.RPCConsumerLogs,
	chainParser ChainParser,
	refererData *RefererData,
	consumerWsSubscriptionManager *ConsumerWSSubscriptionManager,
) (ChainListener, error) {
	switch listenEndpoint.ApiInterface {
	case spectypes.APIInterfaceJsonRPC:
		return NewJrpcChainListener(ctx, listenEndpoint, relaySender, healthReporter, rpcConsumerLogs, refererData, consumerWsSubscriptionManager), nil
	case spectypes.APIInterfaceTendermintRPC:
		return NewTendermintRpcChainListener(ctx, listenEndpoint, relaySender, healthReporter, rpcConsumerLogs, refererData, consumerWsSubscriptionManager), nil
	case spectypes.APIInterfaceRest:
		return NewRestChainListener(ctx, listenEndpoint, relaySender, healthReporter, rpcConsumerLogs, refererData), nil
	case spectypes.APIInterfaceGrpc:
		return NewGrpcChainListener(ctx, listenEndpoint, relaySender, healthReporter, rpcConsumerLogs, chainParser, refererData), nil
	}
	return nil, fmt.Errorf("chainListener for apiInterface (%s) not found", listenEndpoint.ApiInterface)
}

type ChainParser interface {
	ParseMsg(url string, data []byte, connectionType string, metadata []pairingtypes.Metadata, extensionInfo extensionslib.ExtensionInfo) (ChainMessage, error)
	SetSpec(spec spectypes.Spec)
	DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32)
	ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData, blocksInFinalizationProof uint32)
	GetParsingByTag(tag spectypes.FUNCTION_TAG) (parsing *spectypes.ParseDirective, apiCollection *spectypes.ApiCollection, existed bool)
	CraftMessage(parser *spectypes.ParseDirective, connectionType string, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error)
	HandleHeaders(metadata []pairingtypes.Metadata, apiCollection *spectypes.ApiCollection, headersDirection spectypes.Header_HeaderType) (filtered []pairingtypes.Metadata, overwriteReqBlock string, ignoredMetadata []pairingtypes.Metadata)
	GetVerifications(supported []string) ([]VerificationContainer, error)
	SeparateAddonsExtensions(supported []string) (addons, extensions []string, err error)
	SetPolicy(policy PolicyInf, chainId string, apiInterface string) error
	Active() bool
	Activate()
	UpdateBlockTime(newBlockTime time.Duration)
	GetUniqueName() string
	ExtensionsParser() *extensionslib.ExtensionParser
}

type ChainMessage interface {
	RequestedBlock() (latest int64, earliest int64)
	UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) (modified bool)
	AppendHeader(metadata []pairingtypes.Metadata)
	GetExtensions() []*spectypes.Extension
	OverrideExtensions(extensionNames []string, extensionParser *extensionslib.ExtensionParser)
	DisableErrorHandling()
	TimeoutOverride(...time.Duration) time.Duration
	GetForceCacheRefresh() bool
	SetForceCacheRefresh(force bool) bool
	CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string)

	ChainMessageForSend
}

type ChainMessageForSend interface {
	TimeoutOverride(...time.Duration) time.Duration
	GetApi() *spectypes.Api
	GetRPCMessage() rpcInterfaceMessages.GenericMessage
	GetApiCollection() *spectypes.ApiCollection
	GetParseDirective() *spectypes.ParseDirective
	CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string)
}

type HealthReporter interface {
	IsHealthy() bool
}

type RelaySender interface {
	SendRelay(
		ctx context.Context,
		url string,
		req string,
		connectionType string,
		dappID string,
		consumerIp string,
		analytics *metrics.RelayMetrics,
		metadataValues []pairingtypes.Metadata,
	) (*common.RelayResult, error)
	ParseRelay(
		ctx context.Context,
		url string,
		req string,
		connectionType string,
		dappID string,
		consumerIp string,
		analytics *metrics.RelayMetrics,
		metadata []pairingtypes.Metadata,
	) (ChainMessage, map[string]string, *pairingtypes.RelayPrivateData, error)
	SendParsedRelay(
		ctx context.Context,
		dappID string,
		consumerIp string,
		analytics *metrics.RelayMetrics,
		chainMessage ChainMessage,
		directiveHeaders map[string]string,
		relayRequestData *pairingtypes.RelayPrivateData,
	) (relayResult *common.RelayResult, errRet error)
	CreateDappKey(dappID, consumerIp string) string
	CancelSubscriptionContext(subscriptionKey string)
	SetConsistencySeenBlock(blockSeen int64, key string)
}

type ChainListener interface {
	Serve(ctx context.Context, cmdFlags common.ConsumerCmdFlags)
}

type ChainRouter interface {
	SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend, extensions []string) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, proxyUrl common.NodeUrl, chainId string, err error) // has to be thread safe, reuse code within ParseMsg as common functionality
	ExtensionsSupported([]string) bool
}

type ChainProxy interface {
	GetChainProxyInformation() (common.NodeUrl, string)
	SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) // has to be thread safe, reuse code within ParseMsg as common functionality
}

func GetChainRouter(ctx context.Context, nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint, chainParser ChainParser) (ChainRouter, error) {
	var proxyConstructor func(context.Context, uint, lavasession.RPCProviderEndpoint, ChainParser) (ChainProxy, error)
	switch rpcProviderEndpoint.ApiInterface {
	case spectypes.APIInterfaceJsonRPC:
		proxyConstructor = NewJrpcChainProxy
	case spectypes.APIInterfaceTendermintRPC:
		proxyConstructor = NewtendermintRpcChainProxy
	case spectypes.APIInterfaceRest:
		proxyConstructor = NewRestChainProxy
	case spectypes.APIInterfaceGrpc:
		proxyConstructor = NewGrpcChainProxy
	default:
		return nil, fmt.Errorf("chain proxy for apiInterface (%s) not found", rpcProviderEndpoint.ApiInterface)
	}
	return newChainRouter(ctx, nConns, *rpcProviderEndpoint, chainParser, proxyConstructor)
}
