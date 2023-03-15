package thirdparty

import (
	"context"
	"net/http"

	"github.com/cosmos/cosmos-sdk/server/grpc/gogoreflection"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	cosmos_thirdparty "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/Cosmos"
	cosmwasm "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/Cosmwasm"
	ibc_thirdparty "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/Ibc"
	juno_thirdparty "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/Juno"
	lava_thirdparty "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/Lavanet"
	osmosis_thirdparty "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/Osmosis"
	"github.com/lavanet/lava/utils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

func RegisterServer(chain string, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) (*grpc.Server, http.Server, error) {
	s := grpc.NewServer()
	wrappedServer := grpcweb.WrapServer(s)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		// Set CORS headers
		resp.Header().Set("Access-Control-Allow-Origin", "*")
		resp.Header().Set("Access-Control-Allow-Headers", "Content-Type,x-grpc-web")

		wrappedServer.ServeHTTP(resp, req)
	}

	httpServer := http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{}),
	}

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
	case "JUN1", "JUNT1":
		cosmos_thirdparty.RegisterJunoProtobufs(s, cb)
		ibc_thirdparty.RegisterJunoProtobufs(s, cb)
		juno_thirdparty.RegisterJunoProtobufs(s, cb)
		cosmwasm.RegisterJunoProtobufs(s, cb)
	case "EVMOS", "EVMOST":
		cosmos_thirdparty.RegisterCosmosProtobufs(s, cb)
		ibc_thirdparty.RegisterCosmosProtobufs(s, cb)
		//TODO: add other evmos protobufs missing
	case "CANTO", "CANTOT":
		cosmos_thirdparty.RegisterCosmosProtobufs(s, cb)
		ibc_thirdparty.RegisterCosmosProtobufs(s, cb)
		//TODO: add other canto protobufs missing
	case "AXELAR", "AXELART":
		cosmos_thirdparty.RegisterCosmosProtobufs(s, cb)
		ibc_thirdparty.RegisterCosmosProtobufs(s, cb)
		//TODO: add other canto protobufs missing
	default:
		utils.LavaFormatFatal("Unsupported Chain Server: "+chain, nil, nil)
	}

	utils.LavaFormatInfo("gogoreflection.Register()", nil)
	gogoreflection.Register(s)
	return s, httpServer, nil
}
