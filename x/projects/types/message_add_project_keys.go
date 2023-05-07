package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddProjectKeys = "add_project_keys"

var _ sdk.Msg = &MsgAddProjectKeys{}

func NewMsgAddProjectKeys(creator string, projectID string, projectKeys []ProjectKey) *MsgAddProjectKeys {
	return &MsgAddProjectKeys{
		Creator:     creator,
		Project:     projectID,
		ProjectKeys: projectKeys,
	}
}

func (msg *MsgAddProjectKeys) Route() string {
	return RouterKey
}

func (msg *MsgAddProjectKeys) Type() string {
	return TypeMsgAddProjectKeys
}

func (msg *MsgAddProjectKeys) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddProjectKeys) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddProjectKeys) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	return nil
}
