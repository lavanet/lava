package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgConflictVoteReveal = "conflict_vote_reveal"

var _ sdk.Msg = &MsgConflictVoteReveal{}

func NewMsgConflictVoteReveal(creator string, voteID string, nonce int64, hash []byte) *MsgConflictVoteReveal {
	return &MsgConflictVoteReveal{
		Creator: creator,
		VoteID:  voteID,
		Nonce:   nonce,
		Hash:    hash,
	}
}

func (msg *MsgConflictVoteReveal) Route() string {
	return RouterKey
}

func (msg *MsgConflictVoteReveal) Type() string {
	return TypeMsgConflictVoteReveal
}

func (msg *MsgConflictVoteReveal) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgConflictVoteReveal) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgConflictVoteReveal) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
