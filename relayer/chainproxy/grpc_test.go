package chainproxy

import (
	"context"
	"testing"

	proxypb "github.com/lavanet/lava/relayer/chainproxy/gen/go/proxy/v1"
	"github.com/lavanet/lava/relayer/chainproxy/grpcutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

//nolint:gochecknoglobals // It's fine.
var lavaGRPC = "0.0.0.0:3342" // TODO: os.Getenv("LAVA_GRPC")?

func TestServer_Proxy(t *testing.T) {
	var err error
	ctx := context.Background()

	cl := proxypb.NewProxyServiceClient(grpcutil.MustDial(ctx, lavaGRPC))

	var pbAny *anypb.Any
	pbAny, err = newAny([]byte(`{"height": 20}`))
	require.NoError(t, err)

	var resp *proxypb.ProxyResponse
	resp, err = cl.Proxy(ctx, &proxypb.ProxyRequest{
		Name: "cosmos.base.tendermint.v1beta1.Service/GetLatestBlock",
		Body: pbAny,
	})
	require.NoError(t, err)

	grpcutil.Log(resp.GetBody())
}

func newAny(b []byte) (pbAny *anypb.Any, err error) {
	pbAny = new(anypb.Any)
	if err = grpcutil.Unmarshal(b, pbAny); err != nil {
		return nil, err
	}

	return pbAny, nil
}
