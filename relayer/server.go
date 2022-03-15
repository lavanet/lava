package relayer

import (
	context "context"
	"encoding/json"
	"errors"
	"log"
	"net"

	grpc "google.golang.org/grpc"
)

type relayServer struct {
	UnimplementedRelayerServer
}

var g_conn *Connector

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

	//
	// Unmarshal request
	var msg jsonrpcMessage
	err := json.Unmarshal(in.Data, &msg)
	if err != nil {
		return nil, errors.New("hello")
	}

	//
	// Get node
	rpc, err := g_conn.GetRpc(true)
	if err != nil {
		return nil, errors.New("hello 2")
	}
	defer g_conn.ReturnRpc(rpc)

	//
	// Call our node
	var result json.RawMessage
	err = rpc.CallContext(ctx, &result, msg.Method, msg.Params...)
	if err != nil {
		log.Println(err)
		return nil, errors.New("hello 2")
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
		return nil, errors.New("hello 3")
	}
	reply := RelayReply{
		Data: data,
	}
	return &reply, nil
}

func Server(ctx context.Context, listenAddr string, nodeUrl string) {
	log.Println("Server starting", listenAddr, "node", nodeUrl)

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
