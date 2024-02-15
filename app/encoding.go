package app

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/lavanet/lava/app/params"
)

// makeEncodingConfig creates an EncodingConfig for an amino-based test configuration.
func makeEncodingConfig() (params.EncodingConfig, error) {
	amino := codec.NewLegacyAmino()
	if amino == nil {
		return params.EncodingConfig{}, errors.New("failed to create LegacyAmino codec")
	}

	interfaceRegistry := types.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	return params.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
		TxConfig:          txCfg,
		Amino:             amino,
	}, nil
}

// MakeEncodingConfig creates an EncodingConfig for testing.
func MakeEncodingConfig() (params.EncodingConfig, error) {
	encodingConfig, err := makeEncodingConfig()

	if err != nil {
		return params.EncodingConfig{}, err
	}

	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	return encodingConfig, nil
}
