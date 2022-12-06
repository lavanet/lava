package thirdparty

import (
	"context"

	"github.com/cosmos/cosmos-sdk/server/grpc/gogoreflection"
	cosmos_thirdparty "github.com/lavanet/lava/relayer/chainproxy/thirdparty/cosmos"
	lava_thirdparty "github.com/lavanet/lava/relayer/chainproxy/thirdparty/lava"
	"github.com/lavanet/lava/utils"
	"google.golang.org/grpc"
)

func RegisterServer(chain string, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) (*grpc.Server, error) {

	s := grpc.NewServer()
	cosmos_thirdparty.RegisterCosmosProtobufs(s, cb) // currently all gRPC are cosmos-sdk based. so we need to initialize it.

	switch chain {
	case "LAV1":
		utils.LavaFormatInfo("Chain:"+chain, nil)

		lava_thirdparty.RegisterLavaProtobufs(s, cb)
	default:
		utils.LavaFormatFatal("Unsupported Chain Server: "+chain, nil, nil)
	}

	utils.LavaFormatInfo("gogoreflection.Register()", nil)
	gogoreflection.Register(s)
	return s, nil
}
