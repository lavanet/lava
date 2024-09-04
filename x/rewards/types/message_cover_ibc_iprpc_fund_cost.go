package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgCoverIbcIprpcFundCost = "cover_ibc_iprpc_fund_cost"

var _ sdk.Msg = &MsgCoverIbcIprpcFundCost{}

func NewMsgCoverIbcIprpcFundCost(creator string, index uint64) *MsgCoverIbcIprpcFundCost {
	return &MsgCoverIbcIprpcFundCost{
		Creator: creator,
		Index:   index,
	}
}

func (msg *MsgCoverIbcIprpcFundCost) Route() string {
	return RouterKey
}

func (msg *MsgCoverIbcIprpcFundCost) Type() string {
	return TypeMsgCoverIbcIprpcFundCost
}

func (msg *MsgCoverIbcIprpcFundCost) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCoverIbcIprpcFundCost) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCoverIbcIprpcFundCost) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	return nil
}
