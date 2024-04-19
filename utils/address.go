package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func IsBech32Address(addr string) bool {
	_, err := sdk.AccAddressFromBech32(addr)
	return err == nil
}
