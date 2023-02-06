package cosmwasm_thirdparty

import (
	"context"

	pkg "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/thirdparty_utils/cosmwasm_types"
	"google.golang.org/grpc"
)

func RegisterOsmosisProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	cosmwasmwasmv1 := &implementedCosmwasmWasmV1{cb: cb}
	pkg.RegisterQueryServer(s, cosmwasmwasmv1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterJunoProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	cosmwasmwasmv1 := &implementedCosmwasmWasmV1{cb: cb}
	pkg.RegisterQueryServer(s, cosmwasmwasmv1)

	// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
