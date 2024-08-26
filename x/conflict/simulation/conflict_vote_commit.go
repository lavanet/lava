package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/lavanet/lava/v2/x/conflict/keeper"
	"github.com/lavanet/lava/v2/x/conflict/types"
)

func SimulateMsgConflictVoteCommit(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgConflictVoteCommit{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the ConflictVoteCommit simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "ConflictVoteCommit simulation not implemented"), nil, nil
	}
}
