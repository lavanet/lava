package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

const TypeMsgSetPolicy = "set_admin_policy"

var _ sdk.Msg = &MsgSetPolicy{}

func NewMsgSetPolicy(creator, project string, policy *planstypes.Policy) *MsgSetPolicy {
	return &MsgSetPolicy{
		Creator: creator,
		Project: project,
		Policy:  policy,
	}
}

func (msg *MsgSetPolicy) Route() string {
	return RouterKey
}

func (msg *MsgSetPolicy) Type() string {
	return TypeMsgSetPolicy
}

func (msg *MsgSetPolicy) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSetPolicy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetPolicy) ValidateBasic() error {
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
