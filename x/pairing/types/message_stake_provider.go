package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
)

const TypeMsgStakeProvider = "stake_provider"

var _ sdk.Msg = &MsgStakeProvider{}

func NewMsgStakeProvider(creator, validator, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation int32, delegateLimit sdk.Coin, delegateCommission uint64, provider string, description stakingtypes.Description) *MsgStakeProvider {
	return &MsgStakeProvider{
		Creator:            creator,
		Validator:          validator,
		ChainID:            chainID,
		Amount:             amount,
		Endpoints:          endpoints,
		Geolocation:        geolocation,
		DelegateLimit:      delegateLimit,
		DelegateCommission: delegateCommission,
		Address:            provider,
		Description:        description,
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
	if _, err := sdk.ValAddressFromBech32(msg.Validator); err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "Invalid validator address (%s) %s", err.Error(), msg.Validator)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid provider address (%s)", err)
	}

	if _, err := msg.Description.EnsureLength(); err != nil {
		return sdkerrors.Wrapf(InvalidDescriptionError, "error: %s", err.Error())
	}

	if msg.DelegateCommission > 100 {
		return sdkerrors.Wrapf(DelegateCommissionOOBError, "commission out of bound (%d)", msg.DelegateCommission)
	}

	if err := msg.DelegateLimit.Validate(); err != nil {
		return sdkerrors.Wrapf(DelegateLimitError, "Invalid coin (%s)", err.Error())
	}

	if !msg.Amount.IsValid() {
		return legacyerrors.ErrInvalidCoins
	}

	return nil
}
