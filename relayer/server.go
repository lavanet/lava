package relayer

import (
	context "context"
	"encoding/json"
	"errors"
	"log"
	"net"

	"github.com/ethereum/go-ethereum/rpc"
	grpc "google.golang.org/grpc"
)

type relayServer struct {
	UnimplementedRelayerServer
}

var json_rpc *rpc.Client

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

func (s *relayServer) Relay(ctx context.Context, in *RelayRequest) (*RelayReply, error) {
	log.Println("server got Relay")

	var msg jsonrpcMessage
	err := json.Unmarshal(in.Data, &msg)
	if err != nil {
		return nil, errors.New("hello")
	}

	// TODO: bug bug (wip)
	// can not use global here without locks
	var result string
	err = json_rpc.CallContext(ctx, &result, msg.Method, msg.Params...)
	if err != nil {
		log.Println(err)
		return nil, errors.New("hello 2")
	}
	reply := RelayReply{
		Data: []byte(result),
	}
	return &reply, nil
}

func Server(ctx context.Context, listenAddr string, nodeUrl string) {
	log.Println("Server starting", listenAddr, "node", nodeUrl)

	//
	// Setup client
	rpc, err := rpc.Dial(nodeUrl)
	if err != nil {
		log.Fatalln("Start", err)
	}
	json_rpc = rpc

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
