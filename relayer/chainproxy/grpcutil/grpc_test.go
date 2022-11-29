package grpcutil

//
// import (
// 	"context"
// 	"testing"
//
// 	proxypb "github.com/lavanet/lava/relayer/chainproxy/gen/go/proxy/v1"
// 	"github.com/lavanet/lava/relayer/chainproxy/grpcutil"
// 	"github.com/stretchr/testify/require"
// 	"google.golang.org/protobuf/types/known/anypb"
// 	"google.golang.org/protobuf/types/known/structpb"
// )
//
// //nolint:gochecknoglobals // It's fine.
// var lavaGRPC = "0.0.0.0:3342" // TODO: os.Getenv("LAVA_GRPC")?
//
// func TestServer_Proxy(t *testing.T) {
// 	ctx := context.Background()
// 	cl := proxypb.NewProxyServiceClient(grpcutil.MustDial(ctx, lavaGRPC))
//
// 	// GetLatestBlock(ctx, t, cl)
// 	GetBlockByHeight(ctx, t, cl)
// }
