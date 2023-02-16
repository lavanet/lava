package rewardserver

import (
	"context"

	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type RewardServer struct {
	rewardsTxSender RewardsTxSender
}

type RewardsTxSender interface {
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelayRequest)
}

func (rws *RewardServer) SendNewProof(ctx context.Context, proof *pairingtypes.RelayRequest, epoch uint64, consumerAddr string) (existingCU uint64, updatedWithProof bool) {
	// TODO: implement
	// get the proof for this consumer for this epoch for this session, update the latest proof
	// write to a channel the epoch
	return 0, false
}

func (rws *RewardServer) SendNewDataReliabilityProof(ctx context.Context, dataReliability *pairingtypes.VRFData, epoch uint64, consumerAddr string) {

}

func NewRewardServer(rewardsTxSender RewardsTxSender) *RewardServer {
	//
	rws := &RewardServer{}
	rws.rewardsTxSender = rewardsTxSender
	return rws
}

func (rws *RewardServer) SubscribeStarted(consumer string, epoch uint64, subscribeID string) {
	// hold off reward claims for subscription while this is still active
}

func (rws *RewardServer) SubscribeEnded(consumer string, epoch uint64, subscribeID string) {
	// can collect now
}
