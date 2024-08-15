package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	// this line is used by starport scaffolding # 1
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDelegate{}, "dualstaking/Delegate", nil)
	cdc.RegisterConcrete(&MsgRedelegate{}, "dualstaking/Redelegate", nil)
	cdc.RegisterConcrete(&MsgUnbond{}, "dualstaking/Unbond", nil)
	cdc.RegisterConcrete(&MsgClaimRewards{}, "dualstaking/MsgClaimRewards", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDelegate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRedelegate{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUnbond{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgClaimRewards{},
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
