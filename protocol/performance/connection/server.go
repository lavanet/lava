package connection

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
)

type RelayerConnectionServer struct {
	pairingtypes.UnimplementedRelayerServer
	guid uint64
}

func (rs *RelayerConnectionServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (rs *RelayerConnectionServer) Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
	peerAddress := common.GetIpFromGrpcContext(ctx)
	utils.LavaFormatInfo("received probe", utils.LogAttr("incoming-ip", peerAddress))
	return &pairingtypes.ProbeReply{
		Guid: rs.guid,
	}, nil
}

func (rs *RelayerConnectionServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	return fmt.Errorf("unimplemented")
}
