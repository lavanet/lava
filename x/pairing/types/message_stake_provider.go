package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const TypeMsgStakeProvider = "stake_provider"

var _ sdk.Msg = &MsgStakeProvider{}

func NewMsgStakeProvider(creator string, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation uint64, moniker string) *MsgStakeProvider {
	return &MsgStakeProvider{
		Creator:     creator,
		ChainID:     chainID,
		Amount:      amount,
		Endpoints:   endpoints,
		Geolocation: geolocation,
		Moniker:     moniker,
	}
}

func (msg *MsgStakeProvider) Route() string {
	return RouterKey
}

func (msg *MsgStakeProvider) Type() string {
	return TypeMsgStakeProvider
}

func (msg *MsgStakeProvider) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStakeProvider) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeProvider) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Moniker == "" {
		return sdkerrors.Wrapf(MonikerEmptyError, "invalid moniker (%s)", msg.Moniker)
	}

	if len(msg.Moniker) > MAX_LEN_MONIKER {
		return sdkerrors.Wrapf(MonikerTooLongError, "invalid moniker (%s)", msg.Moniker)
	}

	return nil
}
