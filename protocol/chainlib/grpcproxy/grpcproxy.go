package grpcproxy

import (
	"context"
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/utils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ProxyCallBack = func(ctx context.Context, method string, reqBody []byte) ([]byte, metadata.MD, error)

func NewGRPCProxy(cb ProxyCallBack) (*grpc.Server, *http.Server, error) {
	s := grpc.NewServer(grpc.UnknownServiceHandler(makeProxyFunc(cb)), grpc.ForceServerCodec(RawBytesCodec{}))
	wrappedServer := grpcweb.WrapServer(s)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		// Set CORS headers
		resp.Header().Set("Access-Control-Allow-Origin", "*")
		resp.Header().Set("Access-Control-Allow-Headers", "Content-Type,x-grpc-web")

		wrappedServer.ServeHTTP(resp, req)
	}

	httpServer := &http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{}),
	}
	return s, httpServer, nil
}

func makeProxyFunc(callBack ProxyCallBack) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream) error {
		// currently the callback function does not account for headers.
		methodName, ok := grpc.MethodFromServerStream(stream)
		if !ok {
			return status.Error(codes.Unavailable, "unable to get method name")
		}
		var reqBytes []byte
		err := stream.RecvMsg(&reqBytes)
		if err != nil {
			return err
		}
		respBytes, md, err := callBack(stream.Context(), methodName[1:], reqBytes) // strip first '/' of the method name
		if err != nil {
			return err
		}
		stream.SetHeader(md)
		return stream.SendMsg(respBytes)
	}
}

type RawBytesCodec struct{}

func (RawBytesCodec) Marshal(v interface{}) ([]byte, error) {
	bytes, ok := v.([]byte)
	if !ok {
		return nil, utils.LavaFormatError("cannot encode type", nil, utils.Attribute{Key: "v", Value: v})
	}
	return bytes, nil
}

func (RawBytesCodec) Unmarshal(data []byte, v interface{}) error {
	bufferPtr, ok := v.(*[]byte)
	if !ok {
		return utils.LavaFormatError("cannot decode into type", nil, utils.Attribute{Key: "v", Value: v})
	}
	*bufferPtr = data
	return nil
}

func (RawBytesCodec) Name() string {
	return "lava/grpc-proxy-codec"
}

func (RawBytesCodec) String() string {
	return RawBytesCodec{}.Name()
}
