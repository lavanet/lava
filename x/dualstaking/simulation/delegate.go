package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/lavanet/lava/v2/x/dualstaking/keeper"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
)

func SimulateMsgDelegate(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgDelegate{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the Delegate simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "Delegate simulation not implemented"), nil, nil
	}
}

func SimulateMsgRedelegate(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgRedelegate{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the Redelegate simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "Redelegate simulation not implemented"), nil, nil
	}
}

func SimulateMsgUnbond(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgUnbond{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the Unbond simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "Unbond simulation not implemented"), nil, nil
	}
}
