package ante

import (
	"github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/lavanet/lava/x/spec/keeper"
)

type ExpeditedProposalFilterAnteDecorator struct {
	k *keeper.Keeper
}

func NewExpeditedProposalFilterAnteDecorator(k *keeper.Keeper) types.AnteDecorator {
	return ExpeditedProposalFilterAnteDecorator{k: k}
}

func (e ExpeditedProposalFilterAnteDecorator) AnteHandle(
	ctx types.Context,
	tx types.Tx,
	simulate bool,
	next types.AnteHandler,
) (newCtx types.Context, err error) {
	msgs := tx.GetMsgs()

	for _, msg := range msgs {
		switch m := msg.(type) {
		case *v1.MsgSubmitProposal:
			if m.GetExpedited() {
			}
		}
	}

	return next(ctx, tx, simulate)
}
