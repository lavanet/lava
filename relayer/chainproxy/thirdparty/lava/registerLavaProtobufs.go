package lava_thirdparty

import (
	"context"

	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc"
)

func RegisterLavaProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	uqs := &implementedQueryServer{cb: cb}
	pairingtypes.RegisterQueryServer(s, uqs)
}
