package chainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lavanet/lava/relayer/sentry"
	servicertypes "github.com/lavanet/lava/x/servicer/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  []interface{}   `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type EthereumMessage struct {
	cp         *EthereumChainProxy
	serviceApi *spectypes.ServiceApi
	msg        *jsonrpcMessage
}

type EthereumChainProxy struct {
	conn    *Connector
	nConns  uint
	nodeUrl string
	sentry  *sentry.Sentry
}

func NewEthereumChainProxy(nodeUrl string, nConns uint, sentry *sentry.Sentry) ChainProxy {
	return &EthereumChainProxy{
		nodeUrl: nodeUrl,
		nConns:  nConns,
		sentry:  sentry,
	}
}

func (cp *EthereumChainProxy) Start(ctx context.Context) error {
	cp.conn = NewConnector(ctx, cp.nConns, cp.nodeUrl)
	if cp.conn == nil {
		return errors.New("g_conn == nil")
	}

	return nil
}

func (cp *EthereumChainProxy) getSupportedApi(name string) (*spectypes.ServiceApi, error) {
	if api, ok := cp.sentry.GetSpecApiByName(name); ok {
		if api.Status != "enabled" {
			return nil, errors.New("api is disabled")
		}
		return &api, nil
	}

	return nil, errors.New("api not supported")
}

func (cp *EthereumChainProxy) ParseMsg(data []byte) (NodeMessage, error) {

	//
	// Unmarshal request
	var msg jsonrpcMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}

	//
	// Check api is supported an save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(msg.Method)
	if err != nil {
		return nil, err
	}
	nodeMsg := &EthereumMessage{
		cp:         cp,
		serviceApi: serviceApi,
		msg:        &msg,
	}
	return nodeMsg, nil
}

func (nm *EthereumMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *EthereumMessage) Send(ctx context.Context) (*servicertypes.RelayReply, error) {
	//
	// Get node
	rpc, err := nm.cp.conn.GetRpc(true)
	if err != nil {
		return nil, err
	}
	defer nm.cp.conn.ReturnRpc(rpc)

	//
	// Call our node
	var result json.RawMessage
	err = rpc.CallContext(ctx, &result, nm.msg.Method, nm.msg.Params...)

	//
	// Wrap result back to json
	replyMsg := jsonrpcMessage{
		Version: nm.msg.Version,
		ID:      nm.msg.ID,
	}
	if err != nil {
		//
		// TODO: CallContext is limited, it does not give us the source
		// of the error or the error code if json (we need smarter error handling)
		replyMsg.Error = &jsonError{
			Code:    1, // TODO
			Message: fmt.Sprintf("%s", err),
		}
	} else {
		replyMsg.Result = result
	}

	data, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, err
	}
	reply := &servicertypes.RelayReply{
		Data: data,
	}
	return reply, nil
}
