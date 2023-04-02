package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddProjectKeys{}, "projects/AddProjectKeys", nil)
	cdc.RegisterConcrete(&MsgSetAdminPolicy{}, "projects/SetAdminPolicy", nil)
	cdc.RegisterConcrete(&MsgSetSubscriptionPolicy{}, "projects/SetSubscriptionPolicy", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddProjectKeys{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetAdminPolicy{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetSubscriptionPolicy{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
