package relayer

import (
	context "context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/x/spec/types"
	grpc "google.golang.org/grpc"
)

var (
	g_conn           *Connector
	g_privKey        *btcSecp256k1.PrivateKey
	g_serverSpec     types.Spec
	g_serverApis     map[string]types.ServiceApi
	g_sessions       map[string]map[uint64]interface{}
	g_sessions_mutex sync.Mutex
)

type relayServer struct {
	UnimplementedRelayerServer
}

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

func isAuthorizedUser(in *RelayRequest) bool {
	pubKey, err := recoverPubKeyFromRelay(in)
	if err != nil {
		log.Println("error: recoverPubKeyFromRelay", err)
		return false
	}

	//
	// TODO: missing pairing check
	log.Println("user addr", pubKey.Address())

	return true
}

func isSupportedSpec(in *RelayRequest) bool {
	return uint64(in.SpecId) == g_serverSpec.Id
}

func getSupportedApi(name string) (*types.ServiceApi, error) {
	if api, ok := g_serverApis[name]; ok {
		return &api, nil
	}

	return nil, errors.New("api not supported")
}

func (s *relayServer) Relay(ctx context.Context, in *RelayRequest) (*RelayReply, error) {
	log.Println("server got Relay")

	//
	//
	if !isAuthorizedUser(in) {
		return nil, errors.New("user not authorized or bad signature")
	}
	if !isSupportedSpec(in) {
		return nil, errors.New("spec not supported by server")
	}

	//
	// Unmarshal request
	var msg jsonrpcMessage
	err := json.Unmarshal(in.Data, &msg)
	if err != nil {
		return nil, err
	}

	//
	// TODO: get or create session
	serviceApi, err := getSupportedApi(msg.Method)
	if err != nil {
		return nil, err
	}
	log.Println("serviceApi", serviceApi)

	//
	// Get node
	rpc, err := g_conn.GetRpc(true)
	if err != nil {
		return nil, err
	}
	defer g_conn.ReturnRpc(rpc)

	//
	// Call our node
	var result json.RawMessage
	err = rpc.CallContext(ctx, &result, msg.Method, msg.Params...)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	//
	// Wrap result back to json
	replyMsg := jsonrpcMessage{
		Version: msg.Version,
		ID:      msg.ID,
		Result:  result,
	}
	data, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, err
	}
	reply := RelayReply{
		Data: data,
	}
	sig, err := signRelay(g_privKey, []byte(reply.String()))
	if err != nil {
		return nil, err
	}
	reply.Sig = sig

	return &reply, nil
}

func Server(ctx context.Context, clientCtx client.Context, queryClient types.QueryClient, listenAddr string, nodeUrl string, specId int) {
	//
	// CU (temp TODO: move to service)
	allSpecs, err := queryClient.SpecAll(ctx, &types.QueryAllSpecRequest{})
	if err != nil {
		log.Fatalln("error: queryClient.SpecAll", err)
	}
	if len(allSpecs.Spec) == 0 || len(allSpecs.Spec) <= specId {
		log.Fatalln("error: bad specId or no specs found", specId)
	}
	g_serverSpec = allSpecs.Spec[specId]
	for _, api := range g_serverSpec.Apis {
		g_serverApis[api.Name] = api
	}

	//
	// Info
	log.Println("Server starting", listenAddr, "node", nodeUrl, "spec", g_serverSpec.Name)

	//
	// Keys
	keyName, err := getKeyName(clientCtx)
	if err != nil {
		log.Fatalln("error: getKeyName", err)
	}

	privKey, err := getPrivKey(clientCtx, keyName)
	if err != nil {
		log.Fatalln("error: getPrivKey", err)
	}
	g_privKey = privKey
	serverKey, _ := clientCtx.Keyring.Key(keyName)
	log.Println("Server pubkey", serverKey.GetPubKey().Address())

	//
	// Node
	g_conn = NewConnector(ctx, 1, nodeUrl)
	if g_conn == nil {
		log.Fatalln("g_conn == nil")
	}

	//
	// GRPC
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	go func() {
		select {
		case <-ctx.Done():
			log.Println("server ctx.Done")
			s.Stop()
		}
	}()

	Server := &relayServer{}
	RegisterRelayerServer(s, Server)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
