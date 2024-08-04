package keeper_test

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/x/protocol/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.ProtocolKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.Version, k.Version(ctx))
}

type testStruct struct {
	_ctx    context.Context
	ctx     sdk.Context
	keepers *testkeeper.Keepers
	servers *testkeeper.Servers
}

func NewTestStruct(t *testing.T) *testStruct {
	servers, keepers, _ctx := testkeeper.InitAllKeepers(t)

	_ctx = testkeeper.AdvanceEpoch(_ctx, keepers)
	ctx := sdk.UnwrapSDKContext(_ctx)

	ts := testStruct{
		_ctx:    _ctx,
		ctx:     ctx,
		keepers: keepers,
		servers: servers,
	}

	return &ts
}

func (ts *testStruct) advanceEpoch() {
	ts._ctx = testkeeper.AdvanceEpoch(ts._ctx, ts.keepers)
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)
}

func newVersion(pver, pmin, cver, cmin string) types.Version {
	return types.Version{
		ProviderTarget: pver,
		ProviderMin:    pmin,
		ConsumerTarget: cver,
		ConsumerMin:    cmin,
	}
}

func TestChangeVersion(t *testing.T) {
	ts := NewTestStruct(t)
	keeper := ts.keepers.Protocol

	defaultVersion := newVersion("0.0.1", "0.0.0", "0.0.1", "0.0.0")

	params := keeper.GetParams(ts.ctx)
	require.Equal(t, params.Version, defaultVersion)

	for _, tt := range []struct {
		name    string
		version types.Version
		good    bool
	}{
		{"invalid 1", newVersion("", "1.1.0", "1.1.0", "1.1.0"), false},
		{"invalid 2", newVersion("a", "1.1.0", "1.1.0", "1.1.0"), false},
		{"invalid 2", newVersion("1", "1.1.0", "1.1.0", "1.1.0"), false},
		{"invalid 3", newVersion("1.1", "1.1.0", "1.1.0", "1.1.0"), false},
		{"invalid 4", newVersion("1.1.1.", "1.1.0", "1.1.0", "1.1.0"), false},
		{"invalid 5", newVersion(".1.1", "1.1.0", "1.1.0", "1.1.0"), false},
		{"invalid 6", newVersion("-2.1.1", "1.1.0", "1.1.0", "1.1.0"), false},
		{"default", newVersion("1.0.0", "0.0.0", "1.0.0", "0.0.0"), true},
		{"forward 1", newVersion("2.0.1", "1.0.0", "2.0.1", "1.0.0"), true},
		{"forward 2", newVersion("2.0.1", "2.0.0", "2.0.1", "2.0.0"), true},
		{"partial", newVersion("3.1.0", "2.0.0", "3.1.0", "2.0.0"), true},
		{"backward 1", newVersion("1.0.0", "2.0.0", "1.0.0", "2.0.0"), false},
		{"backward 2", newVersion("3.1.0", "1.0.0", "3.1.0", "1.0.0"), true},
		{"unchanged", newVersion("3.1.0", "2.0.0", "3.1.0", "2.0.0"), true},
		{"min over", newVersion("3.1.0", "4.0.0", "3.1.0", "4.0.0"), false},
		{"mismatch 1", newVersion("3.1.0", "2.0.0", "4.0.0", "2.0.0"), false},
		{"mismatch 2", newVersion("3.1.0", "3.1.0", "3.1.0", "2.0.0"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts.advanceEpoch()

			val, err := json.Marshal(tt.version)
			require.NoError(t, err)

			k, v := string(types.KeyVersion), string(val)
			err = testkeeper.SimulateParamChange(
				ts.ctx, ts.keepers.ParamsKeeper, types.ModuleName, k, v)
			if !tt.good {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			p := keeper.GetParams(ts.ctx)
			require.Equal(t, p.Version, tt.version)

			params = p
		})
	}
}
