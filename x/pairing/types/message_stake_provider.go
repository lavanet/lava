package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const TypeMsgStakeProvider = "stake_provider"

var _ sdk.Msg = &MsgStakeProvider{}

func NewMsgStakeProvider(creator, validator, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation int32, moniker string, delegateLimit sdk.Coin, delegateCommission uint64) *MsgStakeProvider {
	return &MsgStakeProvider{
		Creator:            creator,
		Validator:          validator,
		ChainID:            chainID,
		Amount:             amount,
		Endpoints:          endpoints,
		Geolocation:        geolocation,
		Moniker:            moniker,
		DelegateLimit:      delegateLimit,
		DelegateCommission: delegateCommission,
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
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.Moniker) > MAX_LEN_MONIKER {
		return sdkerrors.Wrapf(MonikerTooLongError, "invalid moniker (%s)", msg.Moniker)
	}

	if msg.DelegateCommission > 100 {
		return sdkerrors.Wrapf(DelegateCommissionOOBError, "commission out of bound (%d)", msg.DelegateCommission)
	}

	if err = msg.DelegateLimit.Validate(); err != nil {
		return sdkerrors.Wrapf(DelegateLimitError, "Invalid coin (%s)", err.Error())
	}

	_, err = sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid validator address (%s)", err)
	}

	if !msg.Amount.IsValid() {
		return legacyerrors.ErrInvalidCoins
	}

	return nil
}
