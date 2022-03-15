package relayer

import (
	context "context"
	"encoding/json"
	"errors"
	"log"
	"net"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	grpc "google.golang.org/grpc"
)

type relayServer struct {
	UnimplementedRelayerServer
}

var g_conn *Connector
var g_privKey *btcSecp256k1.PrivateKey

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

func (s *relayServer) Relay(ctx context.Context, in *RelayRequest) (*RelayReply, error) {
	log.Println("server got Relay")

	//
	//
	if !isAuthorizedUser(in) {
		return nil, errors.New("user not authorized or bad signature")
	}

	//
	// Unmarshal request
	var msg jsonrpcMessage
	err := json.Unmarshal(in.Data, &msg)
	if err != nil {
		return nil, err
	}

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

func Server(ctx context.Context, listenAddr string, nodeUrl string, clientCtx client.Context) {
	log.Println("Server starting", listenAddr, "node", nodeUrl)

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

	g_conn = NewConnector(ctx, 5, nodeUrl)
	if g_conn == nil {
		log.Fatalln("g_conn == nil")
	}

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
