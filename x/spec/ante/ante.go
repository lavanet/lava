package ante

import (
	"github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

type ExpeditedProposalFilterAnteDecorator struct{}

func NewExpeditedProposalFilterAnteDecorator() types.AnteDecorator {
	return ExpeditedProposalFilterAnteDecorator{}
}

func (e ExpeditedProposalFilterAnteDecorator) AnteHandle(ctx types.Context, tx types.Tx, simulate bool, next types.AnteHandler) (newCtx types.Context, err error) {
	// extract the message from the tx
	msgs := tx.GetMsgs()

	for _, msg := range msgs {
		switch m := msg.(type) {
		case *v1.MsgSubmitProposal:
			if m.GetExpedited() {
				// check that all the messages are not in the blacklist
			}
		}
	}

	return next(ctx, tx, simulate)
}
