package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/dualstaking/keeper"
)

// RedelegationFlager sets the dualstaking redelegation flag when needed.
// when the user sends redelegation tx we dont want the hooks to do anything
type RedelegationFlager struct {
	keeper.Keeper
}

func NewRedelegationFlager(dualstaking keeper.Keeper) RedelegationFlager {
	return RedelegationFlager{Keeper: dualstaking}
}

func (rf RedelegationFlager) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	msgs, err := rf.unwrapAuthz(tx.GetMsgs())
	if err != nil {
		return ctx, err
	}

	err = rf.DisableRedelegationHooks(ctx, msgs)
	if err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}

func (rf RedelegationFlager) unwrapAuthz(msgs []sdk.Msg) ([]sdk.Msg, error) {
	var unwrappedMsgs []sdk.Msg
	for _, txMsg := range msgs {
		if authzMsg, ok := txMsg.(*authz.MsgExec); ok {
			azmsgs, err := authzMsg.GetMessages()
			if err != nil {
				return nil, utils.LavaFormatError("could not unwrap authz from msgs", err)
			}
			azmsgs, err = rf.unwrapAuthz(azmsgs)
			if err != nil {
				return nil, err
			}
			unwrappedMsgs = append(unwrappedMsgs, azmsgs...)
		} else {
			unwrappedMsgs = append(unwrappedMsgs, txMsg)
		}
	}

	return unwrappedMsgs, nil
}

func (rf RedelegationFlager) DisableRedelegationHooks(ctx sdk.Context, msgs []sdk.Msg) error {
	redelegations := false
	others := false
	for _, msg := range msgs {
		if _, ok := msg.(*stakingtypes.MsgBeginRedelegate); ok {
			redelegations = true
		} else {
			others = true
		}
	}

	if redelegations && others {
		return utils.LavaFormatWarning("could not disable redelegation hooks", fmt.Errorf("cannot send batch requests with redelegation messages"))
	}

	rf.Keeper.SetDisableDualstakingHook(ctx, redelegations)

	return nil
}
