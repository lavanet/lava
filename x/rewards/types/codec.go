package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// this line is used by starport scaffolding # 1
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	govcodec "github.com/cosmos/cosmos-sdk/x/gov/codec"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSetIprpcData{}, "rewards/MsgSetIprpcData", nil)
	cdc.RegisterConcrete(&MsgFundIprpc{}, "rewards/MsgFundIprpc", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetIprpcData{},
		&MsgFundIprpc{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(Amino)
	// ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

func init() {
	RegisterCodec(Amino)
	cryptocodec.RegisterCrypto(Amino)
	Amino.Seal()

	// allow authz and gov Amino encoding support
	// this can be used to properly serialize MsgGrant, MsgExec
	// and MsgSubmitProposal instances
	sdk.RegisterLegacyAminoCodec(authzcodec.Amino)
	sdk.RegisterLegacyAminoCodec(govcodec.Amino)
}
