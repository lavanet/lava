package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/x/dualstaking/keeper"
	"github.com/lavanet/lava/x/dualstaking/types"
)

// RedelegationFlager sets the GasMeter in the Context and wraps the next AnteHandler with a defer clause
// to recover from any downstream OutOfGas panics in the AnteHandler chain to return an error with information
// on gas provided and gas used.
// CONTRACT: Must be first decorator in the chain
// CONTRACT: Tx must implement GasTx interface
type RedelegationFlager struct {
	keeper.Keeper
}

func NewRedelegationFlager(dualstaking keeper.Keeper) RedelegationFlager {
	return RedelegationFlager{Keeper: dualstaking}
}

func (rf RedelegationFlager) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	que := types.DelegationQue{}
	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		switch msg.(type) {
		case *stakingtypes.MsgBeginRedelegate:
			que.Redelegate()
		case *stakingtypes.MsgCreateValidator:
			que.DelegateUnbond()
		case *stakingtypes.MsgDelegate:
			que.DelegateUnbond()
		case *stakingtypes.MsgUndelegate:
			que.DelegateUnbond()
		case *stakingtypes.MsgCancelUnbondingDelegation:
			que.DelegateUnbond()
		case *types.MsgDelegate:
			que.DelegateUnbond()
		case *types.MsgUnbond:
			que.DelegateUnbond()

		default:
		}
	}
	rf.SetDelegationQue(ctx, que)
	return next(ctx, tx, simulate)
}
