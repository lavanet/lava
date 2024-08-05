package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/x/conflict/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestConsumerConflicts(t *testing.T) {
	keeper, ctx := keepertest.ConflictKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNConflictVote(keeper, ctx, 10)
	for i, msg := range msgs {
		msg.ClientAddress = "client" + strconv.Itoa(i%2)
		keeper.SetConflictVote(ctx, msg)
	}

	for _, tc := range []struct {
		desc              string
		client            string
		expectedConflicts []string
	}{
		{
			desc:              "client0 conflicts",
			client:            "client0",
			expectedConflicts: []string{"0", "2", "4", "6", "8"},
		},
		{
			desc:              "client1 conflicts",
			client:            "client1",
			expectedConflicts: []string{"1", "3", "5", "7", "9"},
		},
		{
			desc:              "invalid client",
			client:            "dummy",
			expectedConflicts: []string{},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := keeper.ConsumerConflicts(wctx, &types.QueryConsumerConflictsRequest{Consumer: tc.client})
			require.NoError(t, err)
			require.ElementsMatch(t, res.Conflicts, tc.expectedConflicts)
		})
	}
}
