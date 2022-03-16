package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (specN *SpecName) ValidateBasic() error {
	if len(specN.Name) > 100 {
		return errors.New("invalid spec name string, length too big")
	}
	return nil
}

type TmpType struct {
	Spec_id    uint
	Api_id     uint
	Session_id SessionID
	CU_sum     uint64
	Data       string
	ClientSig  string
}

func (tt TmpType) verifySignature() bool {
	//TODO: verify the signature against body
	return true
}

func (clientReq *ClientRequest) ParseData(ctx sdk.Context) (*TmpType, error) {
	//TODO: use the protobuf object, read this from the string field
	parseRes := TmpType{Spec_id: 0,
		Api_id:     0,
		Session_id: SessionID{Num: 0},
		CU_sum:     10,
		Data:       "",
		ClientSig:  "123",
	}
	valid := parseRes.verifySignature()
	if !valid {
		return nil, errors.New("invalid client signature on proof of work request")
	}
	return &parseRes, nil
}
