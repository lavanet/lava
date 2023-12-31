package ante

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/gogoproto/proto"
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

	blacklist := e.k.BlacklistedExpeditedMsgs(ctx)

	for _, msg := range msgs {
		switch m := msg.(type) {
		case *v1.MsgSubmitProposal:
			if m.GetExpedited() {
				expeditedMsgs, err := m.GetMsgs()
				if err != nil {
					return ctx, err
				}

				for _, expeditedMsg := range expeditedMsgs {
					if blacklistContainsMsg(blacklist, expeditedMsg) {
						return ctx, fmt.Errorf("expedited proposal contains blacklisted message %s", proto.MessageName(expeditedMsg))
					}
				}
			}
		}
	}

	return next(ctx, tx, simulate)
}

func blacklistContainsMsg(blacklist []string, msg types.Msg) bool {
	msgName := proto.MessageName(msg)
	for _, blacklistedMsg := range blacklist {
		if blacklistedMsg == msgName {
			return true
		}
	}

	return false
}
