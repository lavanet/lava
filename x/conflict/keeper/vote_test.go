package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/rand"
	"github.com/stretchr/testify/require"
)

func (ts *tester) txConflictVoteCommit(msg *conflicttypes.MsgConflictVoteCommit) (*conflicttypes.MsgConflictVoteCommitResponse, error) {
	return ts.Servers.ConflictServer.ConflictVoteCommit(ts.GoCtx, msg)
}

func (ts *tester) txConflictVoteReveal(msg *conflicttypes.MsgConflictVoteReveal) (*conflicttypes.MsgConflictVoteRevealResponse, error) {
	return ts.Servers.ConflictServer.ConflictVoteReveal(ts.GoCtx, msg)
}

func (ts *tester) txConflictDetection(msg *conflicttypes.MsgDetection) (*conflicttypes.MsgDetectionResponse, error) {
	return ts.Servers.ConflictServer.Detection(ts.GoCtx, msg)
}

func (ts *tester) setupForCommit() (string, conflicttypes.MsgDetection, *pairingtypes.RelayReply, *pairingtypes.RelayReply) {
	ts.setupForConflict(ProvidersCount)

	msg, reply1, reply2, err := common.CreateMsgDetectionTest(ts.GoCtx, ts.consumer, ts.providers[0], ts.providers[1], ts.spec)
	require.Nil(ts.T, err)

	_, err = ts.txConflictDetection(msg)
	require.Nil(ts.T, err)

	events := ts.Ctx.EventManager().Events()
	LastEvent := events[len(events)-1]

	var voteID string
	for _, attr := range LastEvent.Attributes {
		if string(attr.Key) == "voteID" {
			voteID = string(attr.GetValue())
		}
	}
	require.NotEmpty(ts.T, voteID)
	return voteID, *msg, reply1, reply2
}

func TestCommit(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	tests := []struct {
		name    string
		creator string
		voteID  string
		valid   bool
	}{
		{"HappyFlow", ts.providers[2].Addr.String(), voteID, true},
		{"NotVoter", ts.providers[0].Addr.String(), voteID, false},
		{"BadVoteID", ts.providers[3].Addr.String(), "BADVOTEID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := conflicttypes.MsgConflictVoteCommit{}
			msg.Creator = tt.creator
			msg.VoteID = tt.voteID

			nonce := rand.Int63()

			replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
			msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

			_, err := ts.txConflictVoteCommit(&msg)
			if tt.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestDoubleCommit(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.txConflictVoteCommit(&msg)
	require.Nil(t, err)
	_, err = ts.txConflictVoteCommit(&msg)
	require.NotNil(t, err)
}

func TestNotVotersProviders(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	// create new provider not in stake
	_, notVoterProvider := ts.AddAccount(common.PROVIDER, 10, 10000)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = notVoterProvider
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	_, err := ts.txConflictVoteCommit(&msg)
	require.NotNil(t, err) // should reject the commit since we are not in the providers list

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = notVoterProvider
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = 0

	_, err = ts.txConflictVoteReveal(&msgReveal)
	require.NotNil(t, err) // should reject the reveal since we are not in the providers list
}

func TestNewVoterOldVote(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	// add a staked provider
	balance := int64(10000)
	_, notVoterProvider := ts.AddAccount(common.PROVIDER, 10, balance)
	err := ts.StakeProvider(notVoterProvider, ts.spec, balance/10)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	// try to vote with the new provider: will be on the next voting list but not in the old one
	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = notVoterProvider
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	_, err = ts.txConflictVoteCommit(&msg)
	require.NotNil(t, err) // should reject the commit since we are not in the providers list

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = notVoterProvider
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = 0

	_, err = ts.txConflictVoteReveal(&msgReveal)
	require.NotNil(t, err) // should reject the reveal since we are not in the providers list
}

func TestCommitAfterDeadline(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	_, err := ts.txConflictVoteCommit(&msg)
	require.NotNil(t, err)
}

func TestReveal(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.txConflictVoteCommit(&msg)
	require.Nil(t, err)

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	tests := []struct {
		name    string
		creator string
		voteID  string
		nonce   int64
		valid   bool
	}{
		{"HappyFlow", ts.providers[2].Addr.String(), voteID, nonce, true},
		{"NotVoter", ts.providers[0].Addr.String(), voteID, nonce, false},
		{"BadVoteID", ts.providers[3].Addr.String(), "BADVOTEID", nonce, false},
		{"DidntCommit", ts.providers[3].Addr.String(), voteID, nonce, false},
		{"BadData", ts.providers[3].Addr.String(), voteID, nonce + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := conflicttypes.MsgConflictVoteReveal{}
			msg.Creator = tt.creator
			msg.VoteID = tt.voteID
			msg.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
			msg.Nonce = tt.nonce

			_, err = ts.txConflictVoteReveal(&msg)
			if tt.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestPreRevealAndDoubleReveal(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.txConflictVoteCommit(&msg)
	require.Nil(t, err)

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = ts.providers[2].Addr.String()
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = nonce

	_, err = ts.txConflictVoteReveal(&msgReveal) // test reveal before commit finished
	require.NotNil(t, err)

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	_, err = ts.txConflictVoteReveal(&msgReveal) // first valid reveal
	require.Nil(t, err)
	_, err = ts.txConflictVoteReveal(&msgReveal) // second reveal, invalid
	require.NotNil(t, err)
}

func TestRevealExpired(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.txConflictVoteCommit(&msg)
	require.Nil(t, err)

	ts.AdvanceEpochs(ts.VotePeriod()*2 + 1)

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = ts.providers[2].Addr.String()
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = nonce

	_, err = ts.txConflictVoteReveal(&msgReveal)
	require.NotNil(t, err)
}

func TestFullMajorityVote(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, relay1 := ts.setupForCommit()

	nonce := rand.Int63()
	// first 2 voters
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgs := make([]conflicttypes.MsgConflictVoteCommit, ProvidersCount)
	for i := 2; i < ProvidersCount-1; i++ {
		msg := conflicttypes.MsgConflictVoteCommit{}
		msg.VoteID = voteID
		msg.Creator = ts.providers[i].Addr.String()
		msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)
		_, err := ts.txConflictVoteCommit(&msg)
		require.Nil(t, err)
		msgs[i] = msg
	}

	// last voter
	replyDataHash = sigs.AllDataHash(relay1, *detection.ResponseConflict.ConflictRelayData1.Request.RelayData)
	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.VoteID = voteID
	msg.Creator = ts.providers[ProvidersCount-1].Addr.String()
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)
	_, err := ts.txConflictVoteCommit(&msg)
	require.Nil(t, err)

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	// vote reveal with all voters
	// first 2 voters
	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.VoteID = voteID
	msgReveal.Nonce = nonce

	for i := 2; i < ProvidersCount-1; i++ {
		msgReveal.Creator = ts.providers[i].Addr.String()
		msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
		_, err = ts.txConflictVoteReveal(&msgReveal)
		require.Nil(t, err)
	}

	// last voter
	msgReveal.Hash = sigs.AllDataHash(relay1, *detection.ResponseConflict.ConflictRelayData1.Request.RelayData)
	msgReveal.Creator = ts.providers[ProvidersCount-1].Addr.String()
	_, err = ts.txConflictVoteReveal(&msgReveal)
	require.Nil(t, err)

	ts.AdvanceEpochs(ts.VotePeriod())

	// end of vote
	_, found := ts.Keepers.Conflict.GetConflictVote(ts.Ctx, voteID)
	require.False(t, found)

	events := ts.Ctx.EventManager().Events()
	LastEvent := events[len(events)-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteResolvedEventName)
}

func TestFullStrongMajorityVote(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, _ := ts.setupForCommit()

	// vote commit with all voters
	msg := conflicttypes.MsgConflictVoteCommit{}

	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	for i := 2; i < ProvidersCount; i++ {
		msg.Creator = ts.providers[i].Addr.String()
		msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)
		_, err := ts.txConflictVoteCommit(&msg)
		require.Nil(t, err)
	}

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	// vote reveal with all voters
	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = nonce

	for i := 2; i < ProvidersCount; i++ {
		msgReveal.Creator = ts.providers[i].Addr.String()
		_, err := ts.txConflictVoteReveal(&msgReveal)
		require.Nil(t, err)
	}

	ts.AdvanceEpochs(ts.VotePeriod())

	_, found := ts.Keepers.Conflict.GetConflictVote(ts.Ctx, voteID)
	require.False(t, found)

	events := ts.Ctx.EventManager().Events()
	LastEvent := events[len(events)-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteResolvedEventName)
}

func TestNoVotersConflict(t *testing.T) {
	ts := newTester(t)
	voteID, _, _, _ := ts.setupForCommit()

	ts.AdvanceEpochs(ts.VotePeriod() + 1)
	ts.AdvanceEpochs(ts.VotePeriod())

	_, found := ts.Keepers.Conflict.GetConflictVote(ts.Ctx, voteID)
	require.False(t, found)

	events := ts.Ctx.EventManager().Events()
	LastEvent := events[len(events)-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteUnresolvedEventName)
}

func TestNoDecisionVote(t *testing.T) {
	ts := newTester(t)
	voteID, detection, relay0, relay1 := ts.setupForCommit()

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.VoteID = voteID
	nonce := rand.Int63()

	// first vote for provider 0
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Creator = ts.providers[2].Addr.String()
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.txConflictVoteCommit(&msg)
	require.Nil(t, err)

	// first vote for provider 1
	replyDataHash = sigs.AllDataHash(relay1, *detection.ResponseConflict.ConflictRelayData1.Request.RelayData)
	msg.Creator = ts.providers[3].Addr.String()
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err = ts.txConflictVoteCommit(&msg)
	require.Nil(t, err)

	// first vote for none
	noneData := "FAKE"
	replyDataHash = sigs.HashMsg([]byte(noneData))
	msg.Creator = ts.providers[4].Addr.String()
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err = ts.txConflictVoteCommit(&msg)
	require.Nil(t, err)

	ts.AdvanceEpochs(ts.VotePeriod() + 1)

	// reveal
	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.VoteID = voteID
	msgReveal.Nonce = nonce

	// reveal vote provider 0
	msgReveal.Creator = ts.providers[2].Addr.String()
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	_, err = ts.txConflictVoteReveal(&msgReveal)
	require.Nil(t, err)

	// reveal vote provider 1
	msgReveal.Creator = ts.providers[3].Addr.String()
	msgReveal.Hash = sigs.AllDataHash(relay1, *detection.ResponseConflict.ConflictRelayData1.Request.RelayData)
	_, err = ts.txConflictVoteReveal(&msgReveal)
	require.Nil(t, err)

	// reveal vote none
	msgReveal.Creator = ts.providers[4].Addr.String()
	msgReveal.Hash = sigs.HashMsg([]byte(noneData))
	_, err = ts.txConflictVoteReveal(&msgReveal)
	require.Nil(t, err)

	ts.AdvanceEpochs(ts.VotePeriod())

	_, found := ts.Keepers.Conflict.GetConflictVote(ts.Ctx, voteID)
	require.False(t, found)

	events := ts.Ctx.EventManager().Events()
	LastEvent := events[len(events)-1]
	require.Equal(t, "lava_"+conflicttypes.ConflictVoteUnresolvedEventName, LastEvent.Type)
}
