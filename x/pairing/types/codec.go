package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgStakeProvider{}, "pairing/StakeProvider", nil)
	cdc.RegisterConcrete(&MsgUnstakeProvider{}, "pairing/UnstakeProvider", nil)
	cdc.RegisterConcrete(&MsgRelayPayment{}, "pairing/RelayPayment", nil)
	cdc.RegisterConcrete(&MsgFreezeProvider{}, "pairing/Freeze", nil)
	cdc.RegisterConcrete(&MsgUnfreezeProvider{}, "pairing/Unfreeze", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgStakeProvider{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUnstakeProvider{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRelayPayment{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgFreezeProvider{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUnfreezeProvider{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
