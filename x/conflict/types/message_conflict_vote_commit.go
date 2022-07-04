package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgConflictVoteCommit = "conflict_vote_commit"

var _ sdk.Msg = &MsgConflictVoteCommit{}

func NewMsgConflictVoteCommit(creator string, voteID string, hash []byte) *MsgConflictVoteCommit {
	return &MsgConflictVoteCommit{
		Creator: creator,
		VoteID:  voteID,
		Hash:    hash,
	}
}

func (msg *MsgConflictVoteCommit) Route() string {
	return RouterKey
}

func (msg *MsgConflictVoteCommit) Type() string {
	return TypeMsgConflictVoteCommit
}

func (msg *MsgConflictVoteCommit) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgConflictVoteCommit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgConflictVoteCommit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
