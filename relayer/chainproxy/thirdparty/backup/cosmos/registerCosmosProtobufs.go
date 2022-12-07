package cosmos_thirdparty

import (
	"context"

	tm "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	"google.golang.org/grpc"
)

func RegisterCosmosProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	uqs := &implementedQueryServer{cb: cb}
	tm.RegisterServiceServer(s, uqs)
}
