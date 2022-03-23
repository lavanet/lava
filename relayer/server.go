package relayer

import (
	context "context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/tendermint/tendermint/libs/bytes"
	grpc "google.golang.org/grpc"
)

var (
	g_conn           *Connector
	g_privKey        *btcSecp256k1.PrivateKey
	g_sessions       map[string]map[uint64]*RelaySession
	g_sessions_mutex sync.Mutex
	g_sentry         *Sentry
	g_serverSpecId   uint64
)

type RelaySession struct {
	CuSum uint64
	Lock  sync.Mutex
}

type relayServer struct {
	UnimplementedRelayerServer
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (err *jsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}
	return err.Message
}

func (err *jsonError) ErrorCode() int {
	return err.Code
}

func (err *jsonError) ErrorData() interface{} {
	return err.Data
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  []interface{}   `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func getRelayUser(in *RelayRequest) (bytes.HexBytes, error) {
	pubKey, err := recoverPubKeyFromRelay(in)
	if err != nil {
		return nil, err
	}

	return pubKey.Address(), nil
}

func isAuthorizedUser(user bytes.HexBytes) bool {
	//
	// TODO: missing pairing check
	log.Println("user addr", user)
	return true
}

func isSupportedSpec(in *RelayRequest) bool {
	return uint64(in.SpecId) == g_serverSpecId
}

func getSupportedApi(name string, sentry *Sentry) (*spectypes.ServiceApi, error) {
	if api, ok := sentry.GetSpecApiByName(name); ok {
		if api.Status != "enabled" {
			return nil, errors.New("api is disabled")
		}
		return &api, nil
	}

	return nil, errors.New("api not supported")
}

func getOrCreateSession(user bytes.HexBytes, sessionId uint64) *RelaySession {
	g_sessions_mutex.Lock()
	defer g_sessions_mutex.Unlock()

	userString := string(user)
	if _, ok := g_sessions[userString]; !ok {
		g_sessions[userString] = map[uint64]*RelaySession{}
	}

	userSessions := g_sessions[userString]
	if _, ok := userSessions[sessionId]; !ok {
		userSessions[sessionId] = &RelaySession{}
	}

	return userSessions[sessionId]
}

func updateSessionCu(sess *RelaySession, serviceApi *spectypes.ServiceApi, in *RelayRequest) error {
	sess.Lock.Lock()
	defer sess.Lock.Unlock()

	log.Println("updateSessionCu", serviceApi.Name, in.SessionId, serviceApi.ComputeUnits, sess.CuSum, in.CuSum)

	//
	// TODO: do we worry about overflow here?
	if sess.CuSum >= in.CuSum {
		return errors.New("bad cu sum")
	}
	if sess.CuSum+serviceApi.ComputeUnits != in.CuSum {
		return errors.New("bad cu sum")
	}

	sess.CuSum = in.CuSum

	// TODO:
	// save relay request here for reward submission at end of session
	return nil
}

func (s *relayServer) Relay(ctx context.Context, in *RelayRequest) (*RelayReply, error) {
	log.Println("server got Relay")

	//
	// Checks
	user, err := getRelayUser(in)
	if err != nil {
		return nil, err
	}
	if !isAuthorizedUser(user) {
		return nil, errors.New("user not authorized or bad signature")
	}
	if !isSupportedSpec(in) {
		return nil, errors.New("spec not supported by server")
	}

	//
	// Unmarshal request
	var msg jsonrpcMessage
	err = json.Unmarshal(in.Data, &msg)
	if err != nil {
		return nil, err
	}

	//
	//
	serviceApi, err := getSupportedApi(msg.Method, g_sentry)
	if err != nil {
		return nil, err
	}
	relaySession := getOrCreateSession(user, in.SessionId)
	updateSessionCu(relaySession, serviceApi, in)

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

	//
	// Wrap result back to json
	replyMsg := jsonrpcMessage{
		Version: msg.Version,
		ID:      msg.ID,
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

func Server(
	ctx context.Context,
	clientCtx client.Context,
	listenAddr string,
	nodeUrl string,
	specId uint64,
) {
	//
	// Start sentry
	sentry := NewSentry(clientCtx, specId, false)
	err := sentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go sentry.Start(ctx)
	for sentry.GetBlockHeight() == 0 {
		time.Sleep(1 * time.Second)
	}
	g_sentry = sentry
	g_sessions = map[string]map[uint64]*RelaySession{}
	g_serverSpecId = specId

	//
	// Info
	log.Println("Server starting", listenAddr, "node", nodeUrl, "spec", sentry.GetSpecName())

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
