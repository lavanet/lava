package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSetSubscriptionPolicy = "set_subscription_policy"

var _ sdk.Msg = &MsgSetSubscriptionPolicy{}

func NewMsgSetSubscriptionPolicy(creator string, projects []string, policy Policy) *MsgSetSubscriptionPolicy {
	return &MsgSetSubscriptionPolicy{
		Creator:  creator,
		Projects: projects,
		Policy:   &policy,
	}
}

func (msg *MsgSetSubscriptionPolicy) Route() string {
	return RouterKey
}

func (msg *MsgSetSubscriptionPolicy) Type() string {
	return TypeMsgSetSubscriptionPolicy
}

func (msg *MsgSetSubscriptionPolicy) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSetSubscriptionPolicy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetSubscriptionPolicy) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	err = ValidateBasicPolicy(*msg.GetPolicy())
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidPolicy, "invalid policy")
	}
	return nil
}
