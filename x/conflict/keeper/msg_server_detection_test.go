package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	conflictconstruct "github.com/lavanet/lava/x/conflict/types/construct"
	"github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

const ProvidersCount = 5

type tester struct {
	common.Tester
	consumer  sigs.Account
	providers []sigs.Account
	plan      plantypes.Plan
	spec      spectypes.Spec
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}
	val, _ := ts.AddAccount(common.VALIDATOR, 0, 1000000)
	ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(10000)))

	ts.AddPlan("free", common.CreateMockPlan())
	ts.AddSpec("mock", common.CreateMockSpec())

	ts.AdvanceEpoch()

	return ts
}

func (ts *tester) setupForConflict(providersCount int) *tester {
	var (
		balance int64 = 100000
		stake   int64 = 1000
	)

	ts.plan = ts.Plan("free")
	ts.spec = ts.Spec("mock")

	consumer, consumerAddr := ts.AddAccount("consumer", 0, balance)
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, ts.plan.Index, 1, false, false)
	require.Nil(ts.T, err)
	ts.consumer = consumer

	for i := 0; i < providersCount; i++ {
		providerAcct, provider := ts.AddAccount(common.PROVIDER, i, balance)
		err := ts.StakeProvider(providerAcct.GetVaultAddr(), provider, ts.spec, stake)
		require.Nil(ts.T, err)
		ts.providers = append(ts.providers, providerAcct)
	}

	ts.AdvanceEpoch()
	return ts
}

func TestDetection(t *testing.T) {
	ts := newTester(t)
	ts.setupForConflict(ProvidersCount)

	tests := []struct {
		name           string
		Creator        sigs.Account
		Provider0      sigs.Account
		Provider1      sigs.Account
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
		{"HappyFlow", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "", []byte{}, 0, 100, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"CuSumChange", ts.consumer, ts.providers[0], ts.providers[2], "", "", 0, "", []byte{}, 0, 0, 100, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"RelayNumChange", ts.consumer, ts.providers[0], ts.providers[3], "", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"SessionIDChange", ts.consumer, ts.providers[0], ts.providers[4], "", "", 0, "", []byte{}, 0, 0, 0, 1, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), true},
		{"QoSNil", ts.consumer, ts.providers[2], ts.providers[3], "", "", 0, "", []byte{}, 0, 0, 0, 0, nil, []byte("DIFF"), true},
		{"BadCreator", ts.providers[4], ts.providers[0], ts.providers[1], "", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadConnectionType", ts.consumer, ts.providers[0], ts.providers[1], "DIFF", "", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadURL", ts.consumer, ts.providers[0], ts.providers[1], "", "DIFF", 0, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadBlockHeight", ts.consumer, ts.providers[0], ts.providers[1], "", "", 10, "", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadChainID", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "DIFF", []byte{}, 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadData", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "", []byte("DIFF"), 0, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"BadRequestBlock", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "", []byte{}, 10, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte("DIFF"), false},
		{"SameReplyData", ts.consumer, ts.providers[0], ts.providers[1], "", "", 0, "", []byte{}, 10, 0, 0, 0, &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}, []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, _, reply, err := common.CreateResponseConflictMsgDetectionForTest(ts.GoCtx, tt.Creator, tt.Provider0, tt.Provider1, &ts.spec)
			require.NoError(t, err)

			msg.Creator = tt.Creator.Addr.String()

			// changes to request1 according to test
			responseConflict := msg.GetResponseConflict()
			responseConflict.ConflictRelayData1.Request.RelayData.ConnectionType += tt.ConnectionType
			responseConflict.ConflictRelayData1.Request.RelayData.ApiUrl += tt.ApiUrl
			responseConflict.ConflictRelayData1.Request.RelaySession.Epoch += tt.BlockHeight
			responseConflict.ConflictRelayData1.Request.RelaySession.SpecId += tt.ChainID
			responseConflict.ConflictRelayData1.Request.RelayData.Data = append(responseConflict.ConflictRelayData1.Request.RelayData.Data, tt.Data...)
			responseConflict.ConflictRelayData1.Request.RelayData.RequestBlock += tt.RequestBlock
			responseConflict.ConflictRelayData1.Request.RelaySession.CuSum += tt.Cusum
			responseConflict.ConflictRelayData1.Request.RelaySession.QosReport = tt.QoSReport
			responseConflict.ConflictRelayData1.Request.RelaySession.RelayNum += tt.RelayNum
			responseConflict.ConflictRelayData1.Request.RelaySession.SessionId += tt.SeassionID
			responseConflict.ConflictRelayData1.Request.RelaySession.Provider = tt.Provider1.Addr.String()

			responseConflict.ConflictRelayData1.Request.RelaySession.Sig = []byte{}
			sig, err := sigs.Sign(ts.consumer.SK, *responseConflict.ConflictRelayData1.Request.RelaySession)
			require.NoError(t, err)
			responseConflict.ConflictRelayData1.Request.RelaySession.Sig = sig

			reply.Data = append(reply.Data, tt.ReplyData...)
			relayExchange := types.NewRelayExchange(*responseConflict.ConflictRelayData1.Request, *reply)
			sig, err = sigs.Sign(tt.Provider1.SK, relayExchange)
			require.NoError(t, err)
			reply.Sig = sig

			relayFinalization := conflicttypes.NewRelayFinalizationFromRelaySessionAndRelayReply(responseConflict.ConflictRelayData1.Request.RelaySession, reply, ts.consumer.Addr)
			sigBlocks, err := sigs.Sign(tt.Provider1.SK, relayFinalization)
			require.NoError(t, err)
			reply.SigBlocks = sigBlocks

			responseConflict.ConflictRelayData1.Reply = conflictconstruct.ConstructReplyMetadata(reply, responseConflict.ConflictRelayData1.Request)
			// send detection msg
			_, err = ts.txConflictDetection(msg)
			if tt.Valid {
				events := ts.Ctx.EventManager().Events()
				require.NoError(t, err)
				require.Equal(t, events[len(events)-1].Type, utils.EventPrefix+conflicttypes.ConflictVoteDetectionEventName)
			}
		})
	}
}

// TestFrozenProviderDetection checks that frozen providers are not part of the voters in conflict detection
func TestFrozenProviderDetection(t *testing.T) {
	ts := newTester(t)
	ts.setupForConflict(4) // stake 4 providers

	// freeze one of the providers
	frozenProviderAcc := ts.providers[3]
	frozenProvider := frozenProviderAcc.Addr.String()

	_, err := ts.Servers.PairingServer.FreezeProvider(ts.GoCtx, &types.MsgFreezeProvider{
		Creator:  frozenProvider,
		ChainIds: []string{ts.spec.Index},
		Reason:   "test",
	})
	require.NoError(t, err)
	ts.AdvanceEpoch() // apply the freeze

	// send a conflict detection TX
	msg, _, _, err := common.CreateResponseConflictMsgDetectionForTest(ts.GoCtx, ts.consumer, ts.providers[0], ts.providers[1], &ts.spec)
	require.NoError(t, err)
	_, err = ts.txConflictDetection(msg)
	require.NoError(t, err)

	// check voters list
	conflictVotes := ts.Keepers.Conflict.GetAllConflictVote(ts.Ctx)
	votes := conflictVotes[0].Votes

	var votersList []string
	for _, vote := range votes {
		votersList = append(votersList, vote.Address)
	}

	// there should be one voter (there is only one that is not part of the conflict + not frozen)
	require.NotEqual(t, 0, len(votersList))

	// the frozen provider should not be part of the voters list
	require.False(t, lavaslices.Contains(votersList, frozenProvider))
}
