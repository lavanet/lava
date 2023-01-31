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

func (rws *RewardServer) ClaimRewardsForProviderSessions(ctx context.Context, providerSessionsWithConsumer *lavasession.ProviderSessionsWithConsumer, epoch uint64, consumerAddr string) {
	// TODO: implement
	return
}

func NewRewardServer(rewardsTxSender RewardsTxSender) *RewardServer {
	rws := &RewardServer{}
	rws.rewardsTxSender = rewardsTxSender
	return rws
}
