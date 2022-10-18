package main

import (
	"context"
	"log"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/lavanet/lava/relayer/chainproxy/grpcutil"
)

const addr = "127.0.0.1:3342"

func main() {
	_ = callGetLatestBlockRequest()
}

func callGetLatestBlockRequest() (err error) {
	ctx := context.Background()
	conn := grpcutil.MustDial(ctx, addr)
	defer func() { _ = conn.Close() }()

	method := "/cosmos.base.tendermint.v1beta1.Service/GetLatestBlock"
	req := &tmservice.GetLatestBlockRequest{}

	var res tmservice.GetLatestBlockResponse
	if err = conn.Invoke(ctx, method, req, &res); err != nil {
		// log.Println(err.Error())
		return err
	}

	// log.Println(res.String())
	return nil
}

func sad() {
	s := ``

	var getLatestBlockResponse tmservice.GetLatestBlockResponse
	if err := getLatestBlockResponse.XXX_Unmarshal([]byte(s)); err != nil {
		log.Println(err.Error())
	}

	log.Println(getLatestBlockResponse.String())
	// jsonpb.UnmarshalString(s, &getLatestBlockResponse)
}
