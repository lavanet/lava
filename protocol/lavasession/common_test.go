package lavasession

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
)

type printGeos []*Endpoint

func (pg printGeos) String() string {
	stringsArr := []string{}
	for _, endp := range pg {
		stringsArr = append(stringsArr, endp.Geolocation.String())
	}
	return strings.Join(stringsArr, ",")
}

func TestGeoOrdering(t *testing.T) {
	pairingEndpoints := []*Endpoint{
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_EU,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_AF,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_AS,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_AU,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_USE,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_USC,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_USW,
		},
	}

	playbook := []struct {
		name          string
		currentGeo    planstypes.Geolocation
		expectedOrder []planstypes.Geolocation
	}{
		{
			name:       "USC",
			currentGeo: planstypes.Geolocation_USC,
			expectedOrder: []planstypes.Geolocation{
				planstypes.Geolocation_USC,
				planstypes.Geolocation_USE,
				planstypes.Geolocation_USW,
				planstypes.Geolocation_EU,
				planstypes.Geolocation_AF,
				planstypes.Geolocation_AS,
				planstypes.Geolocation_AU,
			},
		},
		{
			name:       "USW",
			currentGeo: planstypes.Geolocation_USW,
			expectedOrder: []planstypes.Geolocation{
				planstypes.Geolocation_USW,
				planstypes.Geolocation_USC,
				planstypes.Geolocation_USE,
				planstypes.Geolocation_AU,
				planstypes.Geolocation_EU,
				planstypes.Geolocation_AF,
				planstypes.Geolocation_AS,
			},
		},
		{
			name:       "EU",
			currentGeo: planstypes.Geolocation_EU,
			expectedOrder: []planstypes.Geolocation{
				planstypes.Geolocation_EU,
				planstypes.Geolocation_USE,
				planstypes.Geolocation_AF,
				planstypes.Geolocation_AS,
				planstypes.Geolocation_USC,
			},
		},
	}

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			SortByGeolocations(pairingEndpoints, play.currentGeo)
			printme := printGeos(pairingEndpoints)
			for idx := range play.expectedOrder {
				require.Equal(t, play.expectedOrder[idx].String(), pairingEndpoints[idx].Geolocation.String(), "different order in index %d %s current Geo: %s", idx, printme, play.currentGeo)
			}
		})
	}
}

type RelayerConnectionServer struct {
	pairingtypes.UnimplementedRelayerServer
	guid uint64
}

func (rs *RelayerConnectionServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (rs *RelayerConnectionServer) Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
	// peerAddress := common.GetIpFromGrpcContext(ctx)
	// utils.LavaFormatInfo("received probe", utils.LogAttr("incoming-ip", peerAddress))
	return &pairingtypes.ProbeReply{
		Guid: rs.guid,
	}, nil
}

func (rs *RelayerConnectionServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	return fmt.Errorf("unimplemented")
}

func startServer() (*grpc.Server, net.Listener) {
	listen := ":0"
	lis, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	tlsConfig := GetTlsConfig(NetworkAddressData{})
	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	pairingtypes.RegisterRelayerServer(srv, &RelayerConnectionServer{})
	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Println("test finished:", err)
		}
	}()
	return srv, lis
}

// Note that locally testing compression will probably be out performed by non compressed.
// due to the overhead of compressing it. while global communication should benefit from reduced latency.
func BenchmarkGRPCServer(b *testing.B) {
	srv, lis := startServer()
	address := lis.Addr().String()
	defer srv.Stop()
	defer lis.Close()

	csp := &ConsumerSessionsWithProvider{}
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := csp.ConnectRawClientWithTimeout(ctx, address)
		if err != nil {
			utils.LavaFormatDebug("waiting for grpc server to launch")
			continue
		}
		cancel()
		break
	}

	runBenchmark := func(b *testing.B, opts ...grpc.DialOption) {
		var tlsConf tls.Config
		tlsConf.InsecureSkipVerify = true
		credentials := credentials.NewTLS(&tlsConf)
		opts = append(opts, grpc.WithTransportCredentials(credentials))
		conn, err := grpc.DialContext(context.Background(), address, opts...)
		if err != nil {
			b.Fatalf("failed to dial server: %v", err)
		}
		defer conn.Close()

		client := pairingtypes.NewRelayerClient(conn)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client.Probe(context.Background(), &pairingtypes.ProbeRequest{Guid: 125, SpecId: "EVMOS", ApiInterface: "jsonrpc"})
		}
	}

	b.Run("WithoutCompression", func(b *testing.B) {
		runBenchmark(b)
	})

	b.Run("WithCompression", func(b *testing.B) {
		runBenchmark(b, grpc.WithDefaultCallOptions(
			grpc.UseCompressor(gzip.Name), // Use gzip compression for outgoing messages
		))
	})

	time.Sleep(3 * time.Second)
}
