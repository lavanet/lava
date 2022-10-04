package chainproxy

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
//
// func GetLatestBlock(ctx context.Context, t *testing.T, cl proxypb.ProxyServiceClient) {
// 	resp, err := cl.Proxy(ctx, &proxypb.ProxyRequest{
// 		Name: "cosmos.base.tendermint.v1beta1.Service/GetLatestBlock",
// 		Body: nil,
// 	})
// 	require.NoError(t, err)
//
// 	grpcutil.Log(resp.GetBody())
// }
//
// func GetBlockByHeight(ctx context.Context, t *testing.T, cl proxypb.ProxyServiceClient) {
// 	pbAny, err := newAny([]byte(`{"height": "3"}`))
// 	require.NoError(t, err)
//
// 	var resp *proxypb.ProxyResponse
// 	resp, err = cl.Proxy(ctx, &proxypb.ProxyRequest{
// 		Name: "cosmos.base.tendermint.v1beta1.Service/GetBlockByHeight",
// 		Body: pbAny,
// 	})
// 	require.NoError(t, err)
//
// 	grpcutil.Log(resp.GetBody())
// }
//
// func newAny(b []byte) (pbAny *anypb.Any, err error) {
// 	var v *structpb.Value
// 	v, err = structpb.NewValue(b)
// 	return anypb.New(v)
// }
//
// // func newAny(b []byte) (pbAny *anypb.Any, err error) {
// // 	pbAny = new(anypb.Any)
// // 	if err = grpcutil.Unmarshal(b, pbAny); err != nil {
// // 		return nil, err
// // 	}
// //
// // 	return pbAny, nil
// // }
