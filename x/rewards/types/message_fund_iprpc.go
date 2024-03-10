package types

import (
	fmt "fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgFundIprpc = "fund_iprpc"

var _ sdk.Msg = &MsgFundIprpc{}

func NewMsgFundIprpc(creator string, spec string, duration uint64, amounts sdk.Coins) *MsgFundIprpc {
	return &MsgFundIprpc{
		Creator:  creator,
		Spec:     spec,
		Duration: duration,
		Amounts:  amounts,
	}
}

func (msg *MsgFundIprpc) Route() string {
	return RouterKey
}

func (msg *MsgFundIprpc) Type() string {
	return TypeMsgFundIprpc
}

func (msg *MsgFundIprpc) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgFundIprpc) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgFundIprpc) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	unique := map[string]struct{}{}
	for _, amount := range msg.Amounts {
		if !amount.IsValid() {
			return sdkerrors.Wrap(fmt.Errorf("invalid amount; invalid denom or negative amount. coin: %s", amount.String()), "")
		}
		if _, ok := unique[amount.Denom]; ok {
			return sdkerrors.Wrap(fmt.Errorf("invalid coins, duplicated denom: %s", amount.Denom), "")
		}
		unique[amount.Denom] = struct{}{}
	}

	return nil
}
