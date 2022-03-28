package chainproxy

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/sentry"
	servicertypes "github.com/lavanet/lava/x/servicer/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type CosmosMessage struct {
	cp         *CosmosChainProxy
	serviceApi *spectypes.ServiceApi
	path       string
	msg        []byte
}

type CosmosChainProxy struct {
	nodeUrl string
	sentry  *sentry.Sentry
}

func NewCosmosChainProxy(nodeUrl string, sentry *sentry.Sentry) ChainProxy {
	nodeUrl = strings.TrimSuffix(nodeUrl, "/")
	return &CosmosChainProxy{
		nodeUrl: nodeUrl,
		sentry:  sentry,
	}
}

func (cp *CosmosChainProxy) GetSentry() *sentry.Sentry {
	return cp.sentry
}

func (cp *CosmosChainProxy) Start(context.Context) error {
	return nil
}

func (cp *CosmosChainProxy) getSupportedApi(path string) (*spectypes.ServiceApi, error) {
	//
	// TODO: this will not find apis with params '{param}' in path
	//
	if api, ok := cp.sentry.GetSpecApiByName(path); ok {
		if api.Status != "enabled" {
			return nil, errors.New("api is disabled")
		}
		return &api, nil
	}

	return nil, errors.New("api not supported")
}

func (cp *CosmosChainProxy) ParseMsg(path string, data []byte) (NodeMessage, error) {
	//
	// Check api is supported an save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(path)
	if err != nil {
		return nil, err
	}
	nodeMsg := &CosmosMessage{
		cp:         cp,
		serviceApi: serviceApi,
		path:       path,
		msg:        data,
	}

	return nodeMsg, nil
}

func (nm *CosmosChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	return
}

func (nm *CosmosMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *CosmosMessage) Send(ctx context.Context) (*servicertypes.RelayReply, error) {
	httpClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	//
	// TODO: some APIs use POST!
	msgBuffer := bytes.NewBuffer(nm.msg)
	req, err := http.NewRequest(http.MethodGet, nm.cp.nodeUrl+nm.path, msgBuffer)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	reply := &servicertypes.RelayReply{
		Data: body,
	}
	return reply, nil
}
