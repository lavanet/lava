package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/lavanet/lava/v4/x/subscription/keeper"
	"github.com/lavanet/lava/v4/x/subscription/types"
)

func SimulateMsgDelProject(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgDelProject{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the DelProject simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "DelProject simulation not implemented"), nil, nil
	}
}
