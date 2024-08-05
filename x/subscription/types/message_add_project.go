package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
)

const TypeMsgAddProject = "add_project"

var _ sdk.Msg = &MsgAddProject{}

func NewMsgAddProject(creator string, projectData projectstypes.ProjectData) *MsgAddProject {
	return &MsgAddProject{
		Creator:     creator,
		ProjectData: projectData,
	}
}

func (msg *MsgAddProject) Route() string {
	return RouterKey
}

func (msg *MsgAddProject) Type() string {
	return TypeMsgAddProject
}

func (msg *MsgAddProject) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddProject) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddProject) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	for _, key := range msg.ProjectData.ProjectKeys {
		_, err := sdk.AccAddressFromBech32(key.Key)
		if err != nil {
			return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid project key address (%s)", err)
		}
	}

	return nil
}
