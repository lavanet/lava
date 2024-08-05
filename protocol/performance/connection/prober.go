package connection

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Prober struct {
	address       string
	conn          *grpc.ClientConn
	relayerClient *pairingtypes.RelayerClient
}

func NewProber(addrss string) *Prober {
	return &Prober{address: addrss}
}

func createConnection(ctx context.Context, address string) (*pairingtypes.RelayerClient, *grpc.ClientConn, error) {
	cswp := lavasession.ConsumerSessionsWithProvider{}
	return cswp.ConnectRawClientWithTimeout(ctx, address)
}

func (p *Prober) RunOnce(ctx context.Context) error {
	if p.address == "" {
		return fmt.Errorf("can't run with address empty")
	}
	if p.conn == nil || p.relayerClient == nil {
		relayer, conn, err := createConnection(ctx, p.address)
		if err != nil {
			return err
		}
		p.relayerClient = relayer
		p.conn = conn
	}
	relayerClient := *p.relayerClient
	guid := uint64(rand.Int63())

	probeReq := &pairingtypes.ProbeRequest{
		Guid:         guid,
		SpecId:       "prober",
		ApiInterface: "",
	}
	var trailer metadata.MD
	utils.LavaFormatInfo("[+] sending probe", utils.LogAttr("guid", guid))
	probeResp, err := relayerClient.Probe(ctx, probeReq, grpc.Trailer(&trailer))
	if err != nil {
		return err
	}
	utils.LavaFormatInfo("probe response", utils.LogAttr("guid", probeResp.Guid))
	return nil
}
