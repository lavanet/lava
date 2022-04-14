package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgStakeProvider{}, "pairing/StakeProvider", nil)
	cdc.RegisterConcrete(&MsgStakeClient{}, "pairing/StakeClient", nil)
	cdc.RegisterConcrete(&MsgUnstakeProvider{}, "pairing/UnstakeProvider", nil)
	cdc.RegisterConcrete(&MsgUnstakeClient{}, "pairing/UnstakeClient", nil)
	cdc.RegisterConcrete(&MsgRelayPayment{}, "pairing/RelayPayment", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgStakeProvider{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgStakeClient{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUnstakeProvider{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUnstakeClient{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRelayPayment{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
