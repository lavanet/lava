package utils

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func ValidateCoins(ctx sdk.Context, denom string, coins sdk.Coin, allowZero bool) error {
	if coins.Denom != denom {
		return LavaFormatWarning(fmt.Sprintf("denomination is not %s", denom),
			sdkerrors.ErrInvalidCoins,
			LogAttr("amount", coins),
		)
	}

	if !allowZero && coins.IsZero() {
		return LavaFormatWarning("invalid coin amount: got 0",
			sdkerrors.ErrInvalidCoins, LogAttr("amount", coins))
	}

	if !coins.IsValid() {
		return LavaFormatWarning("invalid coins to delegate",
			sdkerrors.ErrInvalidCoins,
			LogAttr("amount", coins),
		)
	}
	return nil
}
