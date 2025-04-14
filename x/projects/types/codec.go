package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	govcodec "github.com/cosmos/cosmos-sdk/x/gov/codec"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddKeys{}, "projects/AddKeys", nil)
	cdc.RegisterConcrete(&MsgDelKeys{}, "projects/DelKeys", nil)
	cdc.RegisterConcrete(&MsgSetPolicy{}, "projects/SetPolicy", nil)
	cdc.RegisterConcrete(&MsgSetSubscriptionPolicy{}, "projects/SetSubscriptionPolicy", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddKeys{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDelKeys{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetPolicy{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetSubscriptionPolicy{},
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

	// allow authz and gov Amino encoding support
	// this can be used to properly serialize MsgGrant, MsgExec
	// and MsgSubmitProposal instances
	RegisterCodec(authzcodec.Amino)
	RegisterCodec(govcodec.Amino)

	cryptocodec.RegisterCrypto(Amino)
	Amino.Seal()
}
