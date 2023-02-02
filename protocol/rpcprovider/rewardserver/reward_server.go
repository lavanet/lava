package rewardserver

import (
	"context"

	"github.com/lavanet/lava/relayer/lavasession"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type RewardServer struct {
	rewardsTxSender RewardsTxSender
}

type RewardsTxSender interface {
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelayRequest)
}

func (rws *RewardServer) SendNewProof(ctx context.Context, singleProviderSession *lavasession.SingleProviderSession, epoch uint64, consumerAddr string) {
	// TODO: implement
	// get the proof for this consumer for this epoch for this session, update the latest proof
	// write to a channel the epoch
}

func NewRewardServer(rewardsTxSender RewardsTxSender) *RewardServer {
	//
	rws := &RewardServer{}
	rws.rewardsTxSender = rewardsTxSender
	return rws
}
