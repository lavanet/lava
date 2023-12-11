package types

import (
	fmt "fmt"
	"math/big"
	"reflect"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Function to validate a plan object fields
func (p Plan) ValidatePlan() error {
	if p.GetIndex() == "" {
		return sdkerrors.Wrap(ErrInvalidPlanIndex, "plan's index can't be empty")
	}

	// check that the plan's price is non-zero
	if p.GetPrice().Amount.IsZero() {
		return sdkerrors.Wrap(ErrInvalidPlanPrice, "plan's price can't be zero")
	}

	// check that if overuse is allowed then the overuse rate is non-zero
	if p.GetAllowOveruse() && (p.GetOveruseRate().IsNil() || p.GetOveruseRate().IsZero()) {
		return sdkerrors.Wrap(ErrInvalidPlanOveruse, "plan can't allow CU overuse and have overuse rate of zero")
	}

	// check that if overuse is not allowed then the overuse rate is zero
	if !p.GetAllowOveruse() && !(p.GetOveruseRate().IsNil() || p.GetOveruseRate().IsZero()) {
		return sdkerrors.Wrap(ErrInvalidPlanOveruse, "plan can't forbid CU overuse and have a non-zero overuse rate")
	}

	// check that the plan's description length is below the max length
	if len(p.GetDescription()) > MAX_LEN_PLAN_DESCRIPTION {
		return sdkerrors.Wrap(ErrInvalidPlanDescription, "plan's description is too long")
	}

	// check that the plan's type length is below the max length
	if len(p.GetType()) > MAX_LEN_PLAN_TYPE {
		return sdkerrors.Wrap(ErrInvalidPlanType, "plan's type is too long")
	}

	// check that the plan's annual discount is valid
	if p.GetAnnualDiscountPercentage() > uint64(100) {
		return sdkerrors.Wrap(ErrInvalidPlanAnnualDiscount, "plan's annual discount is invalid (not between 0-100 percent)")
	}

	err := p.PlanPolicy.ValidateBasicPolicy(true)
	if err != nil {
		return err
	}

	return nil
}

// PriceDecodeHookFunc helps the decoder to correctly unmarshal the price field's amount (type math.Int)
func PriceDecodeHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t == reflect.TypeOf(sdk.NewInt(0)) {
		amountStr, ok := data.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected data type for amount field")
		}

		// Convert the string amount to an math.Int
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			return nil, fmt.Errorf("failed to convert amount to math.Int")
		}
		return sdk.NewIntFromBigInt(amount), nil
	}

	return data, nil
}

// OveruseRateHookFunc helps the decoder to correctly unmarshal the overuse rate field's amount (type sdk.Coin)
func OveruseRateDecodeHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t == reflect.TypeOf(sdk.Coin{}) && f == reflect.TypeOf("") {
		amountStr, ok := data.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected data type for amount field")
		}

		// Convert the string coins to sdk.Coin
		coins, err := sdk.ParseCoinsNormalized(amountStr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert amount to sdk.Coin: %w", err)
		}

		if len(coins) != 1 {
			return nil, fmt.Errorf("unexpected amount of coins given. Expected: 1 , Got: %d", len(coins))
		}
		return coins[0], nil
	}

	return data, nil
}

var planMandatoryFields = map[string]struct{}{"index": {}, "price": {}}

func CheckPlanMandatoryFields(unsetFields []string) bool {
	planValid := true
	for _, unsetF := range unsetFields {
		if _, ok := planMandatoryFields[unsetF]; ok {
			planValid = false
		}
	}

	return planValid
}
