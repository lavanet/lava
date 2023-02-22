package mock

import "github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"

type MockVoteUpdatable struct {
	epoch uint64
}

func (m MockVoteUpdatable) VoteHandler(*reliabilitymanager.VoteParams, uint64) {
}
