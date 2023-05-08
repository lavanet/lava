package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddKeys = "add_keys"

var _ sdk.Msg = &MsgAddKeys{}

func NewMsgAddKeys(creator string, projectID string, projectKeys []ProjectKey) *MsgAddKeys {
	return &MsgAddKeys{
		Creator:     creator,
		Project:     projectID,
		ProjectKeys: projectKeys,
	}
}

func (msg *MsgAddKeys) Route() string {
	return RouterKey
}

func (msg *MsgAddKeys) Type() string {
	return TypeMsgAddKeys
}

func (msg *MsgAddKeys) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddKeys) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddKeys) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	for _, projectKey := range msg.GetProjectKeys() {
		for _, keyType := range projectKey.GetTypes() {
			if keyType != ProjectKey_ADMIN && keyType != ProjectKey_DEVELOPER {
				return sdkerrors.Wrapf(ErrInvalidKeyType, "project key must be of type ADMIN(=1) or DEVELOPER(=2). projectKey = %d", keyType)
			}
		}
	}
	return nil
}
