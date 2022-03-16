package cli_test

import (
	"fmt"
	"testing"

	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/stretchr/testify/require"
	tmcli "github.com/tendermint/tendermint/libs/cli"
	"google.golang.org/grpc/status"

	"github.com/lavanet/lava/testutil/network"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/servicer/client/cli"
	"github.com/lavanet/lava/x/servicer/types"
)

func networkWithBlockDeadlineForCallbackObjects(t *testing.T) (*network.Network, types.BlockDeadlineForCallback) {
	t.Helper()
	cfg := network.DefaultConfig()
	state := types.GenesisState{}
	require.NoError(t, cfg.Codec.UnmarshalJSON(cfg.GenesisState[types.ModuleName], &state))

	blockDeadlineForCallback := &types.BlockDeadlineForCallback{}
	nullify.Fill(&blockDeadlineForCallback)
	state.BlockDeadlineForCallback = blockDeadlineForCallback
	buf, err := cfg.Codec.MarshalJSON(&state)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg), *state.BlockDeadlineForCallback
}

func TestShowBlockDeadlineForCallback(t *testing.T) {
	net, obj := networkWithBlockDeadlineForCallbackObjects(t)

	ctx := net.Validators[0].ClientCtx
	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	for _, tc := range []struct {
		desc string
		args []string
		err  error
		obj  types.BlockDeadlineForCallback
	}{
		{
			desc: "get",
			args: common,
			obj:  obj,
		},
	} {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			var args []string
			args = append(args, tc.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdShowBlockDeadlineForCallback(), args)
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				var resp types.QueryGetBlockDeadlineForCallbackResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.BlockDeadlineForCallback)
				require.Equal(t,
					nullify.Fill(&tc.obj),
					nullify.Fill(&resp.BlockDeadlineForCallback),
				)
			}
		})
	}
}
