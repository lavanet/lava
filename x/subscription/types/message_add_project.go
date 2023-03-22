package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

const TypeMsgAddProject = "add_project"

var _ sdk.Msg = &MsgAddProject{}

func NewMsgAddProject(creator string, projectName string, enabled bool, consumer string, vrfpk string, projectDescription string) *MsgAddProject {
	return &MsgAddProject{
		Creator:            creator,
		Consumer:           consumer,
		ProjectName:        projectName,
		Enabled:            enabled,
		Vrfpk:              vrfpk,
		ProjectDescription: projectDescription,
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
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Consumer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid consumer address (%s)", err)
	}

	if msg.GetProjectName() == "" || len(msg.GetProjectName()) > projectstypes.MAX_PROJECT_NAME_LEN {
		return sdkerrors.Wrapf(ErrInvalidParameter, "invalid project name (%s). Either empty or too long (max_len = %d)", msg.GetProjectName(), projectstypes.MAX_PROJECT_NAME_LEN)
	}

	if len(msg.GetProjectDescription()) > projectstypes.MAX_PROJECT_DESCRIPTION_LEN {
		return sdkerrors.Wrapf(ErrInvalidParameter, "project description too long (%s). max_len = %d", msg.GetProjectDescription(), projectstypes.MAX_PROJECT_DESCRIPTION_LEN)
	}
	return nil
}
