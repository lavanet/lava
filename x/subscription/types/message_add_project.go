package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	projectstypes "github.com/lavanet/lava/x/projects/types"
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
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	for _, projectKey := range msg.GetProjectData().ProjectKeys {
		_, err = sdk.AccAddressFromBech32(projectKey.GetKey())
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address in project key (%s)", err)
		}

		for _, projectKeyType := range projectKey.Types {
			if projectKeyType != projectstypes.ProjectKey_ADMIN && projectKeyType != projectstypes.ProjectKey_DEVELOPER {
				return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "invalid type in project key (should be 1 or 2)")
			}
		}
	}

	if !projectstypes.ValidateProjectNameAndDescription(msg.GetProjectData().Name, msg.GetProjectData().Description) {
		return sdkerrors.Wrapf(ErrInvalidParameter, "invalid project name/description (name: %s, description: %s). Either name empty, name contains \",\", or name/description long (name_max_len = %d, description_max_len = %d)", msg.GetProjectData().Name, msg.GetProjectData().Description, projectstypes.MAX_PROJECT_NAME_LEN, projectstypes.MAX_PROJECT_DESCRIPTION_LEN)
	}

	if msg.GetProjectData().Policy.MaxProvidersToPair <= 1 {
		return sdkerrors.Wrapf(ErrInvalidParameter, "project maxProvidersToPair field is invalid (maxProvidersToPair = %v). This field must be greater than 1", msg.GetProjectData().Policy.MaxProvidersToPair)
	}

	return nil
}
