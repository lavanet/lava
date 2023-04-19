package thirdparty

import (
	"context"
	"github.com/lavanet/lava/grpcproxy"
	"google.golang.org/grpc"
	"net/http"
)

func RegisterServer(chain string, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) (*grpc.Server, http.Server, error) {
	return grpcproxy.NewGRPCProxy(cb)
}

// TODO: removeme
/*
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

	utils.LavaFormatInfo("Registering Chain:" + chain)
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
	default:
		utils.LavaFormatWarning("Spec ID is not in grpc server registry, setting the default descriptors of cosmos", nil)
		cosmos_thirdparty.RegisterCosmosProtobufs(s, cb)
		ibc_thirdparty.RegisterCosmosProtobufs(s, cb)
	}

	utils.LavaFormatInfo("gogoreflection.Register()")
	gogoreflection.Register(s)
	return s, httpServer, nil
}

*/
