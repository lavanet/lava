package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSetProjectPolicy = "set_project_policy"

var _ sdk.Msg = &MsgSetProjectPolicy{}

func NewMsgSetProjectPolicy(creator string, projectId string, policy Policy) *MsgSetProjectPolicy {
	return &MsgSetProjectPolicy{
		Creator: creator,
		Project: projectId,
		Policy:  &policy,
	}
}

func (msg *MsgSetProjectPolicy) Route() string {
	return RouterKey
}

func (msg *MsgSetProjectPolicy) Type() string {
	return TypeMsgSetProjectPolicy
}

func (msg *MsgSetProjectPolicy) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSetProjectPolicy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetProjectPolicy) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.GetPolicy().GetTotalCuLimit() < msg.GetPolicy().GetEpochCuLimit() {
		return sdkerrors.Wrapf(ErrPolicyBasicValidation, "invalid policy. total_cu_limit (%v) is smaller than epoch_cu_limit (%v)", msg.GetPolicy().GetTotalCuLimit(), msg.Policy.GetEpochCuLimit())
	}
	if msg.GetPolicy().GetMaxProvidersToPair() < 1 {
		return sdkerrors.Wrapf(ErrPolicyBasicValidation, "invalid policy. max providers to pair should be at least 1. current value: %v", msg.GetPolicy().GetMaxProvidersToPair())
	}
	return nil
}
