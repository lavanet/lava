package grpcproxy

import (
	"context"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/utils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const MaxCallRecvMsgSize = 1024 * 1024 * 32

type ProxyCallBack = func(ctx context.Context, method string, reqBody []byte) ([]byte, metadata.MD, error)

type HealthReporter interface {
	IsHealthy() bool
}

func NewGRPCProxy(cb ProxyCallBack, healthCheckPath string, cmdFlags common.ConsumerCmdFlags, healthReporter HealthReporter) (*grpc.Server, *http.Server, error) {
	serverReceiveMaxMessageSize := grpc.MaxRecvMsgSize(MaxCallRecvMsgSize) // setting receive size to 32mb instead of 4mb default
	s := grpc.NewServer(grpc.UnknownServiceHandler(makeProxyFunc(cb)), grpc.ForceServerCodec(RawBytesCodec{}), serverReceiveMaxMessageSize)
	wrappedServer := grpcweb.WrapServer(s)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		// Set CORS headers
		resp.Header().Set("Access-Control-Allow-Origin", cmdFlags.OriginFlag)

		if req.Method == http.MethodOptions {
			resp.Header().Set("Access-Control-Allow-Methods", cmdFlags.MethodsFlag)
			resp.Header().Set("Access-Control-Allow-Headers", cmdFlags.HeadersFlag)
			resp.Header().Set("Access-Control-Allow-Credentials", cmdFlags.CredentialsFlag)
			resp.Header().Set("Access-Control-Max-Age", cmdFlags.CDNCacheDuration)
			resp.WriteHeader(fiber.StatusNoContent)
			resp.Write(make([]byte, 0))
			return
		}

		if healthReporter != nil && req.URL.Path == healthCheckPath && req.Method == http.MethodGet {
			if healthReporter.IsHealthy() {
				resp.WriteHeader(fiber.StatusOK)
				resp.Write([]byte("Healthy"))
			} else {
				resp.WriteHeader(fiber.StatusServiceUnavailable)
				resp.Write([]byte("Unhealthy"))
			}

			return
		}
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

		// Convert metadata keys to lowercase
		lowercaseMD := metadata.New(map[string]string{})
		for k, v := range md {
			lowerKey := strings.ToLower(k)
			lowercaseMD[lowerKey] = v
		}
		md = lowercaseMD

		if err := stream.SetHeader(md); err != nil {
			utils.LavaFormatError("Got error when setting header", err, utils.LogAttr("headers", md))
		}
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
