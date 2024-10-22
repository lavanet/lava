package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/lavanet/lava/v4/x/pairing/keeper"
	"github.com/lavanet/lava/v4/x/pairing/types"
)

func SimulateMsgUnstakeProvider(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgUnstakeProvider{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the UnstakeProvider simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "UnstakeProvider simulation not implemented"), nil, nil
	}
}
