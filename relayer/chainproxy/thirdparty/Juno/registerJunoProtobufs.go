package juno_thirdparty

import (
	"context"

	pb_pkg "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/juno/mint/types"
	"google.golang.org/grpc"
)

func RegisterJunoProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	junomint := &implementedJunoMint{cb: cb}
	pb_pkg.RegisterQueryServer(s, junomint)

	// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
