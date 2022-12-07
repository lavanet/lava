
package router_thirdparty

import (
	"context"

	"google.golang.org/grpc"
)

func RegisterCosmosProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	
routerv1 := &implementedRouterV1{cb: cb}
pkg.RegisterQueryServer(s, routerv1)

// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
