package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgSetVersion = "set_version"

var _ sdk.Msg = &MsgSetVersion{}

func NewMsgSetVersion(authority string, version Version) *MsgSetVersion {
	return &MsgSetVersion{
		Authority: authority,
		Version:   &version,
	}
}

func (msg *MsgSetVersion) Route() string {
	return RouterKey
}

func (msg *MsgSetVersion) Type() string {
	return TypeMsgSetVersion
}

func (msg *MsgSetVersion) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

func (msg *MsgSetVersion) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetVersion) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.Wrap(err, "invalid authority address")
	}

	return nil
}
