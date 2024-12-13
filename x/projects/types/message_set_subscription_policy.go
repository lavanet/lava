package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	planstypes "github.com/lavanet/lava/v4/x/plans/types"
)

const TypeMsgSetSubscriptionPolicy = "set_subscription_policy"

var _ sdk.Msg = &MsgSetSubscriptionPolicy{}

func NewMsgSetSubscriptionPolicy(creator string, projects []string, policy *planstypes.Policy) *MsgSetSubscriptionPolicy {
	return &MsgSetSubscriptionPolicy{
		Creator:  creator,
		Projects: projects,
		Policy:   policy,
	}
}

func (msg *MsgSetSubscriptionPolicy) Route() string {
	return RouterKey
}

func (msg *MsgSetSubscriptionPolicy) Type() string {
	return TypeMsgSetSubscriptionPolicy
}

func (msg *MsgSetSubscriptionPolicy) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Policy != nil {
		if err := msg.Policy.ValidateBasicPolicy(false); err != nil {
			return sdkerrors.Wrapf(err, "invalid policy")
		}
	}

	return nil
}
