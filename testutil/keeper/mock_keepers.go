package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

//account keeper mock
type mockAccountKeeper struct {
}

func (k mockAccountKeeper) GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI {
	return nil
}

func (k mockAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return nil
}

//mock bank keeper
type mockBankKeeper struct {
	balance    map[string]sdk.Coins
	moduleBank map[string]map[string]sdk.Coins
}

func (k *mockBankKeeper) SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return nil
}

func (k *mockBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins := k.balance[addr.String()]
	for i := 0; i < coins.Len(); i++ {
		if coins.GetDenomByIndex(i) == denom {
			return coins[i]
		}
	}
	return sdk.NewCoin(denom, sdk.ZeroInt())
}

func (k *mockBankKeeper) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	//TODO support multiple coins
	if amt.Len() > 1 {
		return fmt.Errorf("mockbankkeeper dont support more than 1 coin")
	}
	coin := amt[0]

	accountCoin := k.GetBalance(ctx, senderAddr, coin.Denom)
	if coin.Amount.GT(accountCoin.Amount) {
		return fmt.Errorf("not enough coins")
	}

	k.balance[senderAddr.String()] = k.balance[senderAddr.String()].Sub(amt)
	if k.moduleBank[recipientModule] == nil {
		k.moduleBank[recipientModule] = make(map[string]sdk.Coins)
		k.moduleBank[recipientModule][senderAddr.String()] = sdk.NewCoins()
	}
	k.moduleBank[recipientModule][senderAddr.String()] = k.moduleBank[recipientModule][senderAddr.String()].Add(amt[0])
	return nil
}

func (k *mockBankKeeper) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	//TODO support multiple coins
	coin := amt[0]

	if module, ok := k.moduleBank[senderModule]; ok {
		if coins, ok := module[recipientAddr.String()]; ok {
			if coins[0].IsGTE(coin) {
				k.balance[recipientAddr.String()] = k.balance[recipientAddr.String()].Add(coin)
				module[recipientAddr.String()] = module[recipientAddr.String()].Sub(amt)
				return nil
			}
		}
	}

	return fmt.Errorf("didnt find staked coins")
}
func (k *mockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	return nil
}

func (k *mockBankKeeper) SetBalance(ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins) error {
	k.balance[addr.String()] = amounts
	return nil
}
