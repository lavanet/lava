package chainlib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type JsonRPCChainParser struct{}

func (apip *JsonRPCChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (apip *JsonRPCChainParser) SetSpec(spec spectypes.Spec) {}

func (apip *JsonRPCChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// TODO
	return false, 0
}

func (apip *JsonRPCChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32, blocksInFinalizationProof uint32) {
	// TODO:
	return 0, 0, 0, 0
}

func NewJrpcChainParser() (chainParser *JsonRPCChainParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type JsonRPCChainListener struct{}

func (apil *JsonRPCChainListener) Serve() {
	// POrtal code replace sendRelay with relaySender.SendRelay()
}

func NewJrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (chainListener *JsonRPCChainListener) {
	// open up server for http implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data

	// save the relay sender

	return nil
}

type JrpcChainProxy struct {
	conn *chainproxy.Connector
}

func NewJrpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint) (ChainProxy, error) {
	cp := &JrpcChainProxy{}
	return cp, cp.start(ctx, nConns, rpcProviderEndpoint.NodeUrl)
}

func (cp *JrpcChainProxy) start(ctx context.Context, nConns uint, nodeUrl string) error {
	cp.conn = chainproxy.NewConnector(ctx, nConns, nodeUrl)
	if cp.conn == nil {
		return errors.New("g_conn == nil")
	}

	return nil
}

func (cp *JrpcChainProxy) SendNodeMsg(ctx context.Context, path string, data []byte, connectionType string, ch chan interface{}, chainMessage ChainMessage) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get node
	rpc, err := cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, err
	}
	defer cp.conn.ReturnRpc(rpc)
	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(chainproxy.JsonrpcMessage)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in jsonrpc failed to cast RPCInput from chainMessage", nil, &map[string]string{"rpcMessage": fmt.Sprintf("%+v", rpcInputMessage)})
	}
	// Call our node
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *chainproxy.JsonrpcMessage
	var sub *rpcclient.ClientSubscription
	if ch != nil {
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nodeMessage.ID, nodeMessage.Method, ch, nodeMessage.Params)
	} else {
		connectCtx, cancel := context.WithTimeout(ctx, LocalNodeTimePerCu(chainMessage.GetServiceApi().ComputeUnits))
		defer cancel()
		rpcMessage, err = rpc.CallContext(connectCtx, nodeMessage.ID, nodeMessage.Method, nodeMessage.Params)
	}

	var replyMsg chainproxy.JsonrpcMessage
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		replyMsg = chainproxy.JsonrpcMessage{
			Version: nodeMessage.Version,
			ID:      nodeMessage.ID,
		}
		replyMsg.Error = &rpcclient.JsonError{
			Code:    1,
			Message: fmt.Sprintf("%s", err),
		}
		// this later causes returning an error
	} else {
		replyMessage, err = chainproxy.ConvertJsonRPCMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("jsonRPC error", err, nil)
		}
		replyMsg = *replyMessage
	}

	retData, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, "", nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: retData,
	}

	if ch != nil {
		subscriptionID, err = strconv.Unquote(string(replyMsg.Result))
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("Subscription failed", err, nil)
		}
	}

	return reply, subscriptionID, sub, err
}
