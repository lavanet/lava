package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgProofOfWork = "proof_of_work"

var _ sdk.Msg = &MsgProofOfWork{}

func NewMsgProofOfWork(creator string, spec *SpecName, session *SessionID, clientRequest *ClientRequest, workProof *WorkProof, computeUnits uint64, blockOfWork *BlockNum) *MsgProofOfWork {
	return &MsgProofOfWork{
		Creator:       creator,
		Spec:          spec,
		Session:       session,
		ClientRequest: clientRequest,
		WorkProof:     workProof,
		ComputeUnits:  computeUnits,
		BlockOfWork:   blockOfWork,
	}
}

func (msg *MsgProofOfWork) Route() string {
	return RouterKey
}

func (msg *MsgProofOfWork) Type() string {
	return TypeMsgProofOfWork
}

func (msg *MsgProofOfWork) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgProofOfWork) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgProofOfWork) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
