package keeper_test

import (
	"math/rand"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func setupForCommitTests(t *testing.T) (testStruct, string, conflicttypes.MsgDetection) {
	ts := setupForConflictTests(t)

	var msg conflicttypes.MsgDetection
	msg.Creator = ts.consumer.Addr.String()
	//request 0
	msg.ResponseConflict = &conflicttypes.ResponseConflict{ConflictRelayData0: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &types.RelayReply{}}, ConflictRelayData1: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &types.RelayReply{}}}
	msg.ResponseConflict.ConflictRelayData0.Request.ApiId = 0
	msg.ResponseConflict.ConflictRelayData0.Request.ApiUrl = ""
	msg.ResponseConflict.ConflictRelayData0.Request.BlockHeight = sdk.UnwrapSDKContext(ts.ctx).BlockHeight()
	msg.ResponseConflict.ConflictRelayData0.Request.ChainID = ts.spec.Index
	msg.ResponseConflict.ConflictRelayData0.Request.CuSum = 0
	msg.ResponseConflict.ConflictRelayData0.Request.Data = []byte("DUMMYREQUEST")
	msg.ResponseConflict.ConflictRelayData0.Request.Provider = ts.Providers[0].Addr.String()
	msg.ResponseConflict.ConflictRelayData0.Request.QoSReport = &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}
	msg.ResponseConflict.ConflictRelayData0.Request.RelayNum = 1
	msg.ResponseConflict.ConflictRelayData0.Request.SessionId = 1
	msg.ResponseConflict.ConflictRelayData0.Request.RequestBlock = 100
	msg.ResponseConflict.ConflictRelayData0.Request.DataReliability = nil
	msg.ResponseConflict.ConflictRelayData0.Request.Sig = []byte{}

	sig, err := sigs.SignRelay(ts.consumer.SK, *msg.ResponseConflict.ConflictRelayData0.Request)
	require.Nil(t, err)
	msg.ResponseConflict.ConflictRelayData0.Request.Sig = sig

	//request 1
	temp, _ := msg.ResponseConflict.ConflictRelayData0.Request.Marshal()
	msg.ResponseConflict.ConflictRelayData1.Request.Unmarshal(temp)
	msg.ResponseConflict.ConflictRelayData1.Request.Provider = ts.Providers[1].Addr.String()
	msg.ResponseConflict.ConflictRelayData1.Request.Sig = []byte{}
	sig, err = sigs.SignRelay(ts.consumer.SK, *msg.ResponseConflict.ConflictRelayData1.Request)
	require.Nil(t, err)
	msg.ResponseConflict.ConflictRelayData1.Request.Sig = sig

	//reply 0
	msg.ResponseConflict.ConflictRelayData0.Reply.Nonce = 10
	msg.ResponseConflict.ConflictRelayData0.Reply.FinalizedBlocksHashes = []byte{}
	msg.ResponseConflict.ConflictRelayData0.Reply.LatestBlock = msg.ResponseConflict.ConflictRelayData0.Request.RequestBlock + int64(ts.spec.FinalizationCriteria)
	msg.ResponseConflict.ConflictRelayData0.Reply.Data = []byte("DUMMYREPLY")
	sig, err = sigs.SignRelayResponse(ts.Providers[0].SK, msg.ResponseConflict.ConflictRelayData0.Reply, msg.ResponseConflict.ConflictRelayData0.Request)
	require.Nil(t, err)
	msg.ResponseConflict.ConflictRelayData0.Reply.Sig = sig
	sigBlocks, err := sigs.SignResponseFinalizationData(ts.Providers[0].SK, msg.ResponseConflict.ConflictRelayData0.Reply, msg.ResponseConflict.ConflictRelayData0.Request, ts.consumer.Addr)
	require.Nil(t, err)
	msg.ResponseConflict.ConflictRelayData0.Reply.SigBlocks = sigBlocks

	//reply 1
	temp, _ = msg.ResponseConflict.ConflictRelayData0.Reply.Marshal()
	msg.ResponseConflict.ConflictRelayData1.Reply.Unmarshal(temp)
	msg.ResponseConflict.ConflictRelayData1.Reply.Data = append(msg.ResponseConflict.ConflictRelayData1.Reply.Data, []byte("DIFF")...)
	sig, err = sigs.SignRelayResponse(ts.Providers[1].SK, msg.ResponseConflict.ConflictRelayData1.Reply, msg.ResponseConflict.ConflictRelayData1.Request)
	require.Nil(t, err)
	msg.ResponseConflict.ConflictRelayData1.Reply.Sig = sig
	sigBlocks, err = sigs.SignResponseFinalizationData(ts.Providers[1].SK, msg.ResponseConflict.ConflictRelayData1.Reply, msg.ResponseConflict.ConflictRelayData1.Request, ts.consumer.Addr)
	require.Nil(t, err)
	msg.ResponseConflict.ConflictRelayData1.Reply.SigBlocks = sigBlocks

	//send detection msg
	_, err = ts.servers.ConflictServer.Detection(ts.ctx, &msg)
	require.Nil(t, err)
	LastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]

	var voteID string
	for _, attr := range LastEvent.Attributes {
		if string(attr.Key) == "voteID" {
			voteID = string(attr.GetValue())
		}
	}
	require.NotEmpty(t, voteID)
	return ts, voteID, msg
}

func TestCommit(t *testing.T) {
	ts, voteID, detection := setupForCommitTests(t)

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
			replyDataHash := sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
			msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

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
	ts, voteID, detection := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)

	require.Nil(t, err)
	_, err = ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.NotNil(t, err)
}

func TestCommitAfterDeadline(t *testing.T) {
	ts, voteID, detection := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.NotNil(t, err)
}

func TestReveal(t *testing.T) {
	ts, voteID, detection := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

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
			msg.Hash = sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
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

func TestDoubleReveal(t *testing.T) {
	ts, voteID, detection := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = ts.Providers[2].Addr.String()
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msgReveal.Nonce = nonce

	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)

	require.Nil(t, err)
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.NotNil(t, err)
}

func TestRevealExpired(t *testing.T) {
	ts, voteID, detection := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.Creator = ts.Providers[2].Addr.String()
	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))*2+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.Creator = ts.Providers[2].Addr.String()
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msgReveal.Nonce = nonce

	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.NotNil(t, err)
}

func TestFullVote(t *testing.T) {
	ts, voteID, detection := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}

	msg.VoteID = voteID

	nonce := rand.Int63()
	replyDataHash := sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)
	for i := 2; i < NUM_OF_PROVIDERS; i++ {
		msg.Creator = ts.Providers[i].Addr.String()
		_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
		require.Nil(t, err)
	}

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.VoteID = voteID
	msgReveal.Hash = sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msgReveal.Nonce = nonce

	for i := 2; i < NUM_OF_PROVIDERS; i++ {
		msgReveal.Creator = ts.Providers[i].Addr.String()
		_, err := ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
		require.Nil(t, err)
	}

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, found := ts.keepers.Conflict.GetConflictVote(sdk.UnwrapSDKContext(ts.ctx), voteID)
	require.False(t, found)

	LastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteResolvedEventName)
}

func TestNoVotersConflict(t *testing.T) {
	ts, voteID, _ := setupForCommitTests(t)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, found := ts.keepers.Conflict.GetConflictVote(sdk.UnwrapSDKContext(ts.ctx), voteID)
	require.False(t, found)

	LastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteUnresolvedEventName)
}

func TestNoDecisionVote(t *testing.T) {
	ts, voteID, detection := setupForCommitTests(t)

	msg := conflicttypes.MsgConflictVoteCommit{}
	msg.VoteID = voteID
	nonce := rand.Int63()

	//first vote for provider 0
	replyDataHash := sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

	msg.Creator = ts.Providers[2].Addr.String()
	_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	//first vote for provider 1
	replyDataHash = sigs.HashMsg(detection.ResponseConflict.ConflictRelayData1.Reply.Data)
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

	msg.Creator = ts.Providers[3].Addr.String()
	_, err = ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	//first vote for none
	noneData := "FAKE"
	replyDataHash = sigs.HashMsg([]byte(noneData))
	msg.Hash = conflicttypes.CommitVoteData(nonce, replyDataHash)

	msg.Creator = ts.Providers[4].Addr.String()
	_, err = ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, &msg)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	//reveal
	msgReveal := conflicttypes.MsgConflictVoteReveal{}
	msgReveal.VoteID = voteID
	msgReveal.Nonce = nonce

	//reveal vote provider 0
	msgReveal.Hash = sigs.HashMsg(detection.ResponseConflict.ConflictRelayData0.Reply.Data)
	msgReveal.Creator = ts.Providers[2].Addr.String()
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.Nil(t, err)

	//reveal vote provider 1
	msgReveal.Hash = sigs.HashMsg(detection.ResponseConflict.ConflictRelayData1.Reply.Data)
	msgReveal.Creator = ts.Providers[3].Addr.String()
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.Nil(t, err)

	//reveal vote none
	msgReveal.Hash = sigs.HashMsg([]byte(noneData))
	msgReveal.Creator = ts.Providers[4].Addr.String()
	_, err = ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, &msgReveal)
	require.Nil(t, err)

	for i := 0; i < int(ts.keepers.Conflict.VotePeriod(sdk.UnwrapSDKContext(ts.ctx)))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	_, found := ts.keepers.Conflict.GetConflictVote(sdk.UnwrapSDKContext(ts.ctx), voteID)
	require.False(t, found)

	LastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]
	require.Equal(t, LastEvent.Type, "lava_"+conflicttypes.ConflictVoteUnresolvedEventName)
}
