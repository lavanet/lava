package utils

import (
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

func IsBech32Address(addr string) bool {
	_, err := sdk.AccAddressFromBech32(addr)
	return err == nil
}

// ParseCLIAddress is used to parse address arguments from CLI. If the address is Bech32
// the function simply returns the argument. If it's not, it tries to fetch it from the keyring
func ParseCLIAddress(clientCtx client.Context, address string) (string, error) {
	if address == "" {
		// empty address --> address = creator
		address = clientCtx.GetFromAddress().String()
	} else {
		if IsBech32Address(address) || address == commontypes.EMPTY_PROVIDER {
			return address, nil
		}

		// address is not a valid Bech32 address --> try to fetch from the keyring
		keyringRecord, err := clientCtx.Keyring.Key(address)
		if err != nil {
			return "", err
		}
		addr, err := keyringRecord.GetAddress()
		if err != nil {
			return "", err
		}
		address = addr.String()
	}
	return address, nil
}
