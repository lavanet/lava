package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/rand"
	"github.com/stretchr/testify/require"
)

func setupForCommitTests(t *testing.T) (testStruct, string, conflicttypes.MsgDetection, *pairingtypes.RelayReply, *pairingtypes.RelayReply) {
	ts := setupForConflictTests(t, NUM_OF_PROVIDERS)

	msg, reply1, reply2, err := common.CreateMsgDetectionTest(ts.ctx, ts.consumer, ts.Providers[0], ts.Providers[1], ts.spec)
	require.Nil(t, err)

	// send detection msg
	_, err = ts.servers.ConflictServer.Detection(ts.ctx, msg)
	require.Nil(t, err)
	LastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]

	var voteID string
	for _, attr := range LastEvent.Attributes {
		if string(attr.Key) == "voteID" {
			voteID = string(attr.GetValue())
		}
	}
	require.NotEmpty(t, voteID)
	return ts, voteID, *msg, reply1, reply2
}

func TestCommit(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	tests := []struct {
		name    string
		creator string
		voteID  string
		valid   bool
	}{
		{"HappyFlow", ts.Providers[2].Addr.String(), voteID, true},
		{"NotVoter", ts.Providers[0].Addr.String(), voteID, false},
		{"BadVoteID", ts.Providers[3].Addr.String(), "BADVOTEID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := conflicttypes.MsgConflictVoteCommit{}
			msg.Creator = tt.creator
			msg.VoteID = tt.voteID

			nonce := rand.Int63()

			replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
			msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

			_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
			if tt.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestDoubleCommit(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)

	require.Nil(t, err)
	_, err = ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.NotNil(t, err)
}

func TestNotVotersProviders(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	var notVoterProvider common.Account
	notVoterProvider.SK, notVoterProvider.Addr = sigs.GenerateFloatingKey() // create new provider not in stake

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = notVoterProvider.Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.NotNil(t, err) // should reject the commit since we are not in the providers list

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = notVoterProvider.Addr.String()
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = 0

	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.NotNil(t, err) // should reject the reveal since we are not in the providers list
}

func TestNewVoterOldVote(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	// add a staked provider
	balance := int64(10000)
	notVoterProvider := common.CreateNewAccount(ts.ctx, *ts.keepers, balance)
	common.StakeAccount(t, ts.ctx, *ts.keepers, *ts.servers, notVoterProvider, ts.spec, balance/10)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// try to vote with the new provider, he will be on the next voting list but not in the old one
	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = notVoterProvider.Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.NotNil(t, err) // should reject the commit since we are not in the providers list

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = notVoterProvider.Addr.String()
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = 0

	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.NotNil(t, err) // should reject the reveal since we are not in the providers list
}

func TestCommitAfterDeadline(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.NotNil(t, err)
}

func TestReveal(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	tests := []struct {
		name    string
		creator string
		voteID  string
		nonce   int64
		valid   bool
	}{
		{"HappyFlow", ts.Providers[2].Addr.String(), voteID, nonce, true},
		{"NotVoter", ts.Providers[0].Addr.String(), voteID, nonce, false},
		{"BadVoteID", ts.Providers[3].Addr.String(), "BADVOTEID", nonce, false},
		{"DidntCommit", ts.Providers[3].Addr.String(), voteID, nonce, false},
		{"BadData", ts.Providers[3].Addr.String(), voteID, nonce + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := conflicttypes.MsgConflictVoteReveal{}
			msg.Creator = tt.creator
			msg.VoteID = tt.voteID
			msg.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
			msg.Nonce = tt.nonce

			_, err := ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msg)
			if tt.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestPreRevealAndDoubleReveal(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = ts.Providers[2].Addr.String()
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = nonce

	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal) // test reveal before commit finished
	require.NotNil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal) // first valid reveal

	require.Nil(t, err)
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal) // second reveal, invalid
	require.NotNil(t, err)
}

func TestRevealExpired(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))*2+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = ts.Providers[2].Addr.String()
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = nonce

	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.NotNil(t, err)
}

func TestFullMajorityVote(t *testing.T) {
	ts, voteID, detection, relay0, relay1 := setupForCommitTests(t)

	nonce := rand.Int63()
	// first 2 voters
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgs := make([]conflicttypes.MsgConflictVoteCommit, NUM_OF_PROVIDERS)
	for i := 2; i < NUM_OF_PROVIDERS-1; i++ {
		msg := conflicttypes.MsgConflictVoteCommit{}
		msg.VoteID = voteID
		msg.Creator = ts.Providers[i].Addr.String()
		msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)
		_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
		require.Nil(t, err)
		msgs[i] = msg
	}

	// last voter
	replyDataHash = sigs.AllDataHash(relay1, *detection.ResponseConflict.ConflictRelayData1.Request.RelayData)
	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.VoteID = voteID
	msg.Creator = ts.Providers[NUM_OF_PROVIDERS-1].Addr.String()
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)
	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// vote reveal with all voters
	// first 2 voters
	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.VoteID = voteID
	msgReveal.Nonce = nonce

	for i := 2; i < NUM_OF_PROVIDERS-1; i++ {
		msgReveal.Creator = ts.Providers[i].Addr.String()
		msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
		_, err := ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
		require.Nil(t, err)
	}

	// last voter
	msgReveal.Hash = sigs.AllDataHash(relay1, *detection.ResponseConflict.ConflictRelayData1.Request.RelayData)
	msgReveal.Creator = ts.Providers[NUM_OF_PROVIDERS-1].Addr.String()
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx))); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// end of vote
	_, found := ts.keepers.Conflict.GetConflictVote(sdk.UnwrapSDKContext(ts.ctx), voteID)
	require.False(t, found)

	events := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()
	LastEvent := events[len(events)-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteResolvedEventName)
}

func TestFullStrongMajorityVote(t *testing.T) {
	ts, voteID, detection, relay0, _ := setupForCommitTests(t)

	// vote commit with all voters
	msg := conflicttypes.MsgConflictVoteCommit{}

	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	for i := 2; i < NUM_OF_PROVIDERS; i++ {
		msg.Creator = ts.Providers[i].Addr.String()
		msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)
		_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
		require.Nil(t, err)
	}

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// vote reveal with all voters
	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msgReveal.Nonce = nonce

	for i := 2; i < NUM_OF_PROVIDERS; i++ {
		msgReveal.Creator = ts.Providers[i].Addr.String()
		_, err := ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
		require.Nil(t, err)
	}

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx))); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, found := ts.keepers.Conflict.GetConflictVote(sdk.UnwrapSDKContext(ts.ctx), voteID)
	require.False(t, found)

	LastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteResolvedEventName)
}

func TestNoVotersConflict(t *testing.T) {
	ts, voteID, _, _, _ := setupForCommitTests(t)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx))); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, found := ts.keepers.Conflict.GetConflictVote(sdk.UnwrapSDKContext(ts.ctx), voteID)
	require.False(t, found)

	LastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteUnresolvedEventName)
}

func TestNoDecisionVote(t *testing.T) {
	ts, voteID, detection, relay0, relay1 := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.VoteID = voteID
	nonce := rand.Int63()

	// first vote for provider 0
	replyDataHash := sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	msg.Creator = ts.Providers[2].Addr.String()
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	// first vote for provider 1
	replyDataHash = sigs.AllDataHash(relay1, *detection.ResponseConflict.ConflictRelayData1.Request.RelayData)
	msg.Creator = ts.Providers[3].Addr.String()
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err = ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	// first vote for none
	noneData := "FAKE"
	replyDataHash = sigs.HashMsg([]byte(noneData))
	msg.Creator = ts.Providers[4].Addr.String()
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash, msg.Creator)

	_, err = ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// reveal
	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.VoteID = voteID
	msgReveal.Nonce = nonce

	// reveal vote provider 0
	msgReveal.Creator = ts.Providers[2].Addr.String()
	msgReveal.Hash = sigs.AllDataHash(relay0, *detection.ResponseConflict.ConflictRelayData0.Request.RelayData)
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.Nil(t, err)

	// reveal vote provider 1
	msgReveal.Creator = ts.Providers[3].Addr.String()
	msgReveal.Hash = sigs.AllDataHash(relay1, *detection.ResponseConflict.ConflictRelayData1.Request.RelayData)
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.Nil(t, err)

	// reveal vote none
	msgReveal.Creator = ts.Providers[4].Addr.String()
	msgReveal.Hash = sigs.HashMsg([]byte(noneData))
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx))); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, found := ts.keepers.Conflict.GetConflictVote(sdk.UnwrapSDKContext(ts.ctx), voteID)
	require.False(t, found)

	events := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()
	LastEvent := events[len(events)-1]
	require.Equal(t, "lava_"+conflicttypes.ConflictVoteUnresolvedEventName, LastEvent.Type)
}
