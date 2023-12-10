package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/x/dualstaking/keeper"
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
	redelegations := false
	others := false
	for _, msg := range tx.GetMsgs() {
		if _, ok := msg.(*stakingtypes.MsgBeginRedelegate); ok {
			redelegations = true
		} else {
			others = true
		}
	}

	if redelegations && others {
		return ctx, fmt.Errorf("cannot send batch requests with redelegation messages")
	}

	keeper.RedelegationFlag = redelegations

	return next(ctx, tx, simulate)
}
