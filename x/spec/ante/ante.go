package ante

import (
	"fmt"
	"strings"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/gogoproto/proto"
	"github.com/lavanet/lava/v2/x/spec/keeper"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ExpeditedProposalFilterAnteDecorator struct {
	k keeper.Keeper
}

func NewExpeditedProposalFilterAnteDecorator(k keeper.Keeper) sdk.AnteDecorator {
	return ExpeditedProposalFilterAnteDecorator{k: k}
}

func (e ExpeditedProposalFilterAnteDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	msgs := tx.GetMsgs()

	for i, msg := range msgs {
		err = e.blockDisallowedExpeditedProposals(ctx, msg)
		if err != nil {
			return ctx, fmt.Errorf("message at index %d contained disallowed proposal: %w", i, err)
		}
	}

	return next(ctx, tx, simulate)
}

func (e ExpeditedProposalFilterAnteDecorator) blockDisallowedExpeditedProposals(ctx sdk.Context, msg sdk.Msg) error {
	switch m := msg.(type) {
	case *v1.MsgSubmitProposal:
		// if it's not expedited we simply exit.
		if !m.Expedited {
			return nil
		}
		// if it's expedited we validate the list of proposal
		// messages.
		propMsgs, err := m.GetMsgs()
		if err != nil {
			return err
		}
		// ensure all proposal exec messages are allowed
		for i, propMsg := range propMsgs {
			err = e.ensureInallowlist(ctx, propMsg)
			if err != nil {
				return fmt.Errorf("proposal exec message at index %d is not allowed to be expedited: %w", i, err)
			}
		}
		// all went fine
		return nil

	// unwrap authz.MsgExec messages to ensure people do not use authz
	// to bypass the filter.
	case *authz.MsgExec:
		authzMsgs, err := m.GetMessages()
		if err != nil {
			return err
		}
		for _, authzMsg := range authzMsgs {
			err = e.blockDisallowedExpeditedProposals(ctx, authzMsg)
			if err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (e ExpeditedProposalFilterAnteDecorator) ensureInallowlist(ctx sdk.Context, msg proto.Message) error {
	allowlist := e.getallowlist(ctx)

	switch m := msg.(type) {
	// if the proposal is the one which executes a legacy content, then
	// we check that the content type is in the allowlist.
	case *v1.MsgExecLegacyContent:
		contentName := getCanonicalProtoNameFromAny(m.Content)
		_, allowed := allowlist[contentName]
		if !allowed {
			return fmt.Errorf("expedited proposal, attempted to execute legacy content %s which is disallowed", contentName)
		}
		return nil
	default:
		msgName := proto.MessageName(m)
		_, allowed := allowlist[msgName]
		if !allowed {
			return fmt.Errorf("expedited proposal, attempted to execute proposal message %s which is disallowed", msgName)
		}
		return nil
	}
}

func (e ExpeditedProposalFilterAnteDecorator) getallowlist(ctx sdk.Context) map[string]struct{} {
	allowlist := e.k.AllowlistedExpeditedMsgs(ctx)
	allowlistSet := make(map[string]struct{}, len(allowlist))
	for _, w := range allowlist {
		allowlistSet[w] = struct{}{}
	}
	return allowlistSet
}

func getCanonicalProtoNameFromAny(any *codectypes.Any) string {
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
