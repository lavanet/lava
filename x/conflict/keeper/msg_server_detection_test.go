package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

const NUM_OF_PROVIDERS = 5

type testStruct struct {
	ctx       context.Context
	keepers   *testkeeper.Keepers
	servers   *testkeeper.Servers
	Providers []common.Account
	spec      spectypes.Spec
	consumer  common.Account
}

func setupForConflictTests(t *testing.T, numOfProviders int) testStruct {
	ts := testStruct{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)
	// init keepers state
	var balance int64 = 100000
	// setup consumer
	ts.consumer = common.CreateNewAccount(ts.ctx, *ts.keepers, balance)

	// setup providers
	for i := 0; i < numOfProviders; i++ {
		ts.Providers = append(ts.Providers, common.CreateNewAccount(ts.ctx, *ts.keepers, balance))
	}

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	var stake int64 = 1000
	// stake consumer
	common.StakeAccount(t, ts.ctx, *ts.keepers, *ts.servers, ts.consumer, ts.spec, stake, false)

	// stake providers
	for _, provider := range ts.Providers {
		common.StakeAccount(t, ts.ctx, *ts.keepers, *ts.servers, provider, ts.spec, stake, true)
	}

	// advance for the staking to be valid
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	return ts
}

func TestDetection(t *testing.T) {
	ts := setupForConflictTests(t, NUM_OF_PROVIDERS)
	tests := []struct {
		name           string
		Creator        common.Account
		Provider0      common.Account
		Provider1      common.Account
		ConnectionType string
		ApiUrl         string
		BlockHeight    int64
		ChainID        string
		Data           []byte
		RequestBlock   int64
		Cusum          uint64
		RelayNum       uint64
		SeassionID     uint64
		QoSReport      *types.QualityOfServiceReport
		ReplyData      []byte
		Valid          bool
	}{
		{"HappyFlow", ts.consumer, ts.Providers[0], ts.Providers[1], "", "", 0, "", []byte{}, 0, 100, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"CuSumChange", ts.consumer, ts.Providers[0], ts.Providers[2], "", "", 0, "", []byte{}, 0, 0, 100, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"RelayNumChange", ts.consumer, ts.Providers[0], ts.Providers[3], "", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"SessionIDChange", ts.consumer, ts.Providers[0], ts.Providers[4], "", "", 0, "", []byte{}, 0, 0, 0, 1, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"QoSNil", ts.consumer, ts.Providers[2], ts.Providers[3], "", "", 0, "", []byte{}, 0, 0, 0, 0, nil, []byte("DIFF"), true},
		{"BadCreator", ts.Providers[4], ts.Providers[0], ts.Providers[1], "", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadConnectionType", ts.consumer, ts.Providers[0], ts.Providers[1], "DIFF", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadURL", ts.consumer, ts.Providers[0], ts.Providers[1], "", "DIFF", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadBlockHeight", ts.consumer, ts.Providers[0], ts.Providers[1], "", "", 10, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadChainID", ts.consumer, ts.Providers[0], ts.Providers[1], "", "", 0, "DIFF", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadData", ts.consumer, ts.Providers[0], ts.Providers[1], "", "", 0, "", []byte("DIFF"), 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadRequestBlock", ts.consumer, ts.Providers[0], ts.Providers[1], "", "", 0, "", []byte{}, 10, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"SameReplyData", ts.consumer, ts.Providers[0], ts.Providers[1], "", "", 0, "", []byte{}, 10, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := common.CreateMsgDetection(ts.ctx, tt.Creator, tt.Provider0, tt.Provider1, ts.spec)
			require.Nil(t, err)

			msg.Creator = tt.Creator.Addr.String()

			// changes to request1 according to test
			msg.ResponseConflict.ConflictRelayData1.Request.RelayData.ConnectionType += tt.ConnectionType
			msg.ResponseConflict.ConflictRelayData1.Request.RelayData.ApiUrl += tt.ApiUrl
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Epoch += tt.BlockHeight
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.SpecId += tt.ChainID
			msg.ResponseConflict.ConflictRelayData1.Request.RelayData.Data = append(msg.ResponseConflict.ConflictRelayData1.Request.RelayData.Data, tt.Data...)
			msg.ResponseConflict.ConflictRelayData1.Request.RelayData.RequestBlock += tt.RequestBlock
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.CuSum += tt.Cusum
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.QosReport = tt.QoSReport
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.RelayNum += tt.RelayNum
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.SessionId += tt.SeassionID
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Provider = tt.Provider1.Addr.String()
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Sig = []byte{}
			sig, err := sigs.SignRelay(ts.consumer.SK, *msg.ResponseConflict.ConflictRelayData1.Request.RelaySession)
			require.Nil(t, err)
			msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Sig = sig

			// changes to reply1 according to test
			msg.ResponseConflict.ConflictRelayData1.Reply.Data = append(msg.ResponseConflict.ConflictRelayData1.Reply.Data, tt.ReplyData...)
			sig, err = sigs.SignRelayResponse(tt.Provider1.SK, msg.ResponseConflict.ConflictRelayData1.Reply, msg.ResponseConflict.ConflictRelayData1.Request)
			require.Nil(t, err)
			msg.ResponseConflict.ConflictRelayData1.Reply.Sig = sig
			sigBlocks, err := sigs.SignResponseFinalizationData(tt.Provider1.SK, msg.ResponseConflict.ConflictRelayData1.Reply, msg.ResponseConflict.ConflictRelayData1.Request, ts.consumer.Addr)
			require.Nil(t, err)
			msg.ResponseConflict.ConflictRelayData1.Reply.SigBlocks = sigBlocks

			// send detection msg
			_, err = ts.servers.ConflictServer.Detection(ts.ctx, &msg)
			if tt.Valid {
				require.Nil(t, err)
				require.Equal(t, sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1].Type, "lava_"+conflicttypes.ConflictVoteDetectionEventName)
			}
		})
	}
}
