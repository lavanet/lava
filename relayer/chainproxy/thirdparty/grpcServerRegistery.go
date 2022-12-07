package thirdparty

import (
	"context"

	"github.com/cosmos/cosmos-sdk/server/grpc/gogoreflection"
	cosmos_thirdparty "github.com/lavanet/lava/relayer/chainproxy/thirdparty/Cosmos"
	cosmwasm "github.com/lavanet/lava/relayer/chainproxy/thirdparty/Cosmwasm"
	ibc_thirdparty "github.com/lavanet/lava/relayer/chainproxy/thirdparty/Ibc"
	juno_thirdparty "github.com/lavanet/lava/relayer/chainproxy/thirdparty/Juno"
	lava_thirdparty "github.com/lavanet/lava/relayer/chainproxy/thirdparty/Lavanet"
	osmosis_thirdparty "github.com/lavanet/lava/relayer/chainproxy/thirdparty/Osmosis"
	router_thirdparty "github.com/lavanet/lava/relayer/chainproxy/thirdparty/Router"
	"github.com/lavanet/lava/utils"
	"google.golang.org/grpc"
)

func RegisterServer(chain string, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) (*grpc.Server, error) {

	s := grpc.NewServer()

	utils.LavaFormatInfo("Registering Chain:"+chain, nil)
	switch chain {
	case "LAV1":
		cosmos_thirdparty.RegisterLavaProtobufs(s, cb)
		lava_thirdparty.RegisterLavaProtobufs(s, cb)
		ibc_thirdparty.RegisterLavaProtobufs(s, cb)
	case "COS3", "COS4":
		cosmos_thirdparty.RegisterOsmosisProtobufs(s, cb)
		ibc_thirdparty.RegisterOsmosisProtobufs(s, cb)
		osmosis_thirdparty.RegisterOsmosisProtobufs(s, cb)
		cosmwasm.RegisterOsmosisProtobufs(s, cb)
	case "COS5", "COS5T":
		cosmos_thirdparty.RegisterCosmosProtobufs(s, cb)
		ibc_thirdparty.RegisterCosmosProtobufs(s, cb)
		router_thirdparty.RegisterCosmosProtobufs(s, cb)
	case "JUN1", "JUNT1":
		cosmos_thirdparty.RegisterJunoProtobufs(s, cb)
		ibc_thirdparty.RegisterJunoProtobufs(s, cb)
		juno_thirdparty.RegisterJunoProtobufs(s, cb)
		cosmwasm.RegisterJunoProtobufs(s, cb)
	default:
		utils.LavaFormatFatal("Unsupported Chain Server: "+chain, nil, nil)
	}

	utils.LavaFormatInfo("gogoreflection.Register()", nil)
	gogoreflection.Register(s)
	return s, nil
}
