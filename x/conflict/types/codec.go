package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDetection{}, "conflict/Detection", nil)
	cdc.RegisterConcrete(&MsgConflictVoteCommit{}, "conflict/ConflictVoteCommit", nil)
	cdc.RegisterConcrete(&MsgConflictVoteReveal{}, "conflict/ConflictVoteReveal", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDetection{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgConflictVoteCommit{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgConflictVoteReveal{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
