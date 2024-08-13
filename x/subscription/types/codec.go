package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	// this line is used by starport scaffolding # 1
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgBuy{}, "subscription/Buy", nil)
	cdc.RegisterConcrete(&MsgAddProject{}, "subscription/AddProject", nil)
	cdc.RegisterConcrete(&MsgDelProject{}, "subscription/DelProject", nil)
	cdc.RegisterConcrete(&MsgAutoRenewal{}, "subscription/AutoRenewal", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgBuy{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddProject{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDelProject{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAutoRenewal{},
	)
	// this line is used by starport scaffolding # 3

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
}
