package ante

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
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

	whitelist := e.k.WhitelistedExpeditedMsgs(ctx)

	if err := validateWhitelistedMsgs(whitelist, msgs); err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}

func validateWhitelistedMsgs(whitelist []string, msgs []types.Msg) error {
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *v1.MsgSubmitProposal:
			if m.GetExpedited() {
				expeditedMsgs, err := m.GetMsgs()
				if err != nil {
					return err
				}

				for _, expeditedMsg := range expeditedMsgs {
					if !whitelistContainsMsg(whitelist, expeditedMsg) {
						return fmt.Errorf("expedited proposal contains blacklisted message %s", proto.MessageName(expeditedMsg))
					}
				}
			}
		case *authz.MsgExec:
			msgs, err := m.GetMessages()
			if err != nil {
				return err
			}

			if err := validateWhitelistedMsgs(whitelist, msgs); err != nil {
				return err
			}
		}
	}

	return nil
}

func whitelistContainsMsg(whitelist []string, msg types.Msg) bool {
	msgName := proto.MessageName(msg)
	for _, blacklistedMsg := range whitelist {
		if blacklistedMsg == msgName {
			return true
		}
	}

	return false
}
