package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterLegacyAminoCodec(_ *codec.LegacyAmino)     {}
func RegisterInterfaces(_ codectypes.InterfaceRegistry) {}
