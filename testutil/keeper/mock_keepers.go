package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	tenderminttypes "github.com/tendermint/tendermint/types"
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

type MockBlockStore struct {
	height int64
}

func (b *MockBlockStore) SetHeight(height int64) {
	b.height = height
}

func (b *MockBlockStore) Base() int64 {
	return 0
}
func (b *MockBlockStore) Height() int64 {
	return b.height
}
func (b *MockBlockStore) Size() int64 {
	return 0
}

func (b *MockBlockStore) LoadBaseMeta() *tenderminttypes.BlockMeta {
	return nil
}
func (b *MockBlockStore) LoadBlockMeta(height int64) *tenderminttypes.BlockMeta {
	return &tenderminttypes.BlockMeta{}
}
func (b *MockBlockStore) LoadBlock(height int64) *tenderminttypes.Block {
	return &tenderminttypes.Block{}
}

func (b *MockBlockStore) SaveBlock(block *tenderminttypes.Block, blockParts *tenderminttypes.PartSet, seenCommit *tenderminttypes.Commit) {
}

func (b *MockBlockStore) PruneBlocks(height int64) (uint64, error) {
	return 0, nil
}

func (b *MockBlockStore) LoadBlockByHash(hash []byte) *tenderminttypes.Block {
	return nil
}
func (b *MockBlockStore) LoadBlockPart(height int64, index int) *tenderminttypes.Part {
	return nil
}

func (b *MockBlockStore) LoadBlockCommit(height int64) *tenderminttypes.Commit {
	return nil
}
func (b *MockBlockStore) LoadSeenCommit(height int64) *tenderminttypes.Commit {
	return nil
}
