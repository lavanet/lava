
package juno_thirdparty

import (
	"context"

	"google.golang.org/grpc"
)

func RegisterJunoProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	
junomint := &implementedJunoMint{cb: cb}
pkg.RegisterQueryServer(s, junomint)

// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
