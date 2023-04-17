package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSetAdminPolicy = "set_admin_policy"

var _ sdk.Msg = &MsgSetAdminPolicy{}

func NewMsgSetAdminPolicy(creator string, project string, policy Policy) *MsgSetAdminPolicy {
	return &MsgSetAdminPolicy{
		Creator: creator,
		Project: project,
		Policy:  policy,
	}
}

func (msg *MsgSetAdminPolicy) Route() string {
	return RouterKey
}

func (msg *MsgSetAdminPolicy) Type() string {
	return TypeMsgSetAdminPolicy
}

func (msg *MsgSetAdminPolicy) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSetAdminPolicy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetAdminPolicy) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	err = msg.GetPolicy().ValidateBasicPolicy()
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidPolicy, "invalid policy")
	}
	return nil
}
