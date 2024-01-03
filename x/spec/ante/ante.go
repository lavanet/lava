package ante

import (
	"fmt"
	types2 "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/gogoproto/proto"
	"github.com/lavanet/lava/x/spec/keeper"
	"google.golang.org/protobuf/reflect/protoreflect"
	"strings"
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
						return fmt.Errorf("expedited proposal contains non whitelisted message %s", proto.MessageName(expeditedMsg))
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

	switch m := msg.(type) {
	case *v1beta1.MsgSubmitProposal:
		msgName = getCanonicalProtoNameFromAny(m.Content)
		url := m.Content.GetTypeUrl()
		name := protoreflect.FullName(url)
		if i := strings.LastIndexByte(url, '/'); i >= 0 {
			name = name[i+len("/"):]
		}
		if !name.IsValid() {
			name = ""
		}

		msgName = string(name)
	}

	return whitelistContainsMsgName(whitelist, msgName)
}

func whitelistContainsMsgName(whitelist []string, msgName string) bool {
	for _, whitelistedMsg := range whitelist {
		if whitelistedMsg == msgName {
			return true
		}
	}

	return false
}

func getCanonicalProtoNameFromAny(any *types2.Any) string {
	url := any.GetTypeUrl()
	name := protoreflect.FullName(url)
	if i := strings.LastIndexByte(url, '/'); i >= 0 {
		name = name[i+len("/"):]
	}
	if !name.IsValid() {
		name = ""
	}

	return string(name)
}
