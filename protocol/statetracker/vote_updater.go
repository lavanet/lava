package statetracker

import (
	"sync"

	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"golang.org/x/net/context"
)

const (
	CallbackKeyForVoteUpdate = "vote-update"
)

type VoteUpdatable interface {
	VoteHandler(*reliabilitymanager.VoteParams, uint64) error
}

type VoteUpdater struct {
	lock           sync.RWMutex
	voteUpdatables map[string]*VoteUpdatable
	eventTracker   *EventTracker
}

func NewVoteUpdater(eventTracker *EventTracker) *VoteUpdater {
	return &VoteUpdater{voteUpdatables: map[string]*VoteUpdatable{}, eventTracker: eventTracker}
}

func (vu *VoteUpdater) RegisterVoteUpdatable(ctx context.Context, voteUpdatable *VoteUpdatable, endpoint lavasession.RPCEndpoint) {
	vu.lock.Lock()
	defer vu.lock.Unlock()
	vu.voteUpdatables[endpoint.Key()] = voteUpdatable
}

func (vu *VoteUpdater) UpdaterKey() string {
	return CallbackKeyForVoteUpdate
}

func (vu *VoteUpdater) Update(latestBlock int64) {
	vu.lock.RLock()
	defer vu.lock.RUnlock()
	votes, err := vu.eventTracker.getLatestVoteEvents()
	if err != nil {
		return
	}
	for _, vote := range votes {
		endpoint := lavasession.RPCEndpoint{ChainID: vote.ChainID, ApiInterface: vote.ApiInterface}
		updatable := vu.voteUpdatables[endpoint.Key()]
		if updatable == nil {
			continue
		}
		(*updatable).VoteHandler(vote, uint64(latestBlock))
	}
}
