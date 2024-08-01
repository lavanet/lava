package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

const TypeMsgClaimRewards = "claim_rewards"

var _ sdk.Msg = &MsgClaimRewards{}

func NewMsgClaimRewards(delegator string, provider string) *MsgClaimRewards {
	return &MsgClaimRewards{
		Creator:  delegator,
		Provider: provider,
	}
}

func (msg *MsgClaimRewards) Route() string {
	return RouterKey
}

func (msg *MsgClaimRewards) Type() string {
	return TypeMsgClaimRewards
}

func (msg *MsgClaimRewards) GetSigners() []sdk.AccAddress {
	delegator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{delegator}
}

func (msg *MsgClaimRewards) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgClaimRewards) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid delegator address (%s)", err)
	}

	if msg.Provider != "" && msg.Provider != commontypes.EMPTY_PROVIDER {
		_, err = sdk.AccAddressFromBech32(msg.Provider)
		if err != nil {
			return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid provider address (%s)", err)
		}
	}

	return nil
}
