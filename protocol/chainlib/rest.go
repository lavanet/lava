package chainlib

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type RestChainParser struct{}

func (apip *RestChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (apip *RestChainParser) SetSpec(spec spectypes.Spec) {}

func (apip *RestChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// TODO
	return false, 0
}

func (apip *RestChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32, blocksInFinalizationProof uint32) {
	// TODO:
	return 0, 0, 0, 0
}

func NewRestChainParser() (chainParser *RestChainParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type RestChainListener struct{}

func (apil *RestChainListener) Serve() {}

func NewRestChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (chainListener *RestChainListener) {
	// open up server for http implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data
	return nil
}

type RestChainProxy struct {
	nodeUrl string
}

func NewRestChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint) (ChainProxy, error) {
	nodeUrl := strings.TrimSuffix(rpcProviderEndpoint.NodeUrl, "/")
	rcp := &RestChainProxy{nodeUrl: nodeUrl}
	return rcp, nil
}

func (rcp *RestChainProxy) SendNodeMsg(ctx context.Context, path string, data []byte, connectionType string, ch chan interface{}, chainMessage ChainMessage) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	if ch != nil {
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on rest", nil, nil)
	}
	httpClient := http.Client{
		Timeout: LocalNodeTimePerCu(chainMessage.GetServiceApi().ComputeUnits),
	}

	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(chainproxy.RestMessage)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in rest, failed to cast RPCInput from chainMessage", nil, &map[string]string{"rpcMessage": fmt.Sprintf("%+v", rpcInputMessage)})
	}

	var connectionTypeSlected string = http.MethodGet
	// if ConnectionType is default value or empty we will choose http.MethodGet otherwise choosing the header type provided
	if chainMessage.GetInterface().Type != "" {
		connectionTypeSlected = chainMessage.GetInterface().Type
	}

	msgBuffer := bytes.NewBuffer(nodeMessage)
	url := rcp.nodeUrl + nodeMessage.Path
	// Only get calls uses query params the rest uses the body
	if connectionTypeSlected == http.MethodGet {
		url += string(nodeMessage.Msg)
	}
	req, err := http.NewRequest(connectionTypeSlected, url, msgBuffer)
	if err != nil {
		return nil, "", nil, err
	}

	// setting the content-type to be application/json instead of Go's defult http.DefaultClient
	if connectionTypeSlected == http.MethodPost || connectionTypeSlected == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, "", nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, "", nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: body,
	}
	return reply, "", nil, nil
}
