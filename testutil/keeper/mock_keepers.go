package keeper

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	tenderminttypes "github.com/tendermint/tendermint/types"
)

// account keeper mock
type mockAccountKeeper struct{}

func (k mockAccountKeeper) GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI {
	return nil
}

func (k mockAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return sdk.AccAddress([]byte(moduleName))
}

// mock bank keeper
type mockBankKeeper struct {
	balance map[string]sdk.Coins
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
	// TODO support multiple coins
	moduleAcc := sdk.AccAddress([]byte(recipientModule))
	if amt.Len() > 1 {
		return fmt.Errorf("mockbankkeeper dont support more than 1 coin")
	}
	coin := amt[0]

	accountCoin := k.GetBalance(ctx, senderAddr, coin.Denom)
	if coin.Amount.GT(accountCoin.Amount) {
		return fmt.Errorf("not enough coins")
	}

	k.SubFromBalance(senderAddr, amt)
	k.AddToBalance(moduleAcc, amt)
	return nil
}

func (k *mockBankKeeper) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	// TODO support multiple coins

	moduleAcc := sdk.AccAddress([]byte(senderModule))

	if amt.Len() > 1 {
		return fmt.Errorf("mockbankkeeper doesn't support more than 1 coin")
	}
	coin := amt[0]

	accountCoin := k.GetBalance(ctx, moduleAcc, coin.Denom)
	if coin.Amount.GT(accountCoin.Amount) {
		return fmt.Errorf("not enough coins")
	}

	k.SubFromBalance(moduleAcc, amt)

	k.AddToBalance(recipientAddr, amt)

	return nil
}

func (k *mockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	acc := sdk.AccAddress([]byte(moduleName))
	k.AddToBalance(acc, amounts)
	return nil
}

func (k *mockBankKeeper) SetBalance(ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins) error {
	k.balance[addr.String()] = amounts
	return nil
}

func (k *mockBankKeeper) AddToBalance(addr sdk.AccAddress, amounts sdk.Coins) error {
	if _, ok := k.balance[addr.String()]; ok {
		for _, coin := range amounts {
			k.balance[addr.String()] = k.balance[addr.String()].Add(coin)
		}
	} else {
		k.balance[addr.String()] = amounts
	}
	return nil
}

func (k *mockBankKeeper) SubFromBalance(addr sdk.AccAddress, amounts sdk.Coins) error {
	if _, ok := k.balance[addr.String()]; ok {
		k.balance[addr.String()] = k.balance[addr.String()].Sub(amounts)
	} else {
		return fmt.Errorf("acount is empty, can't sub")
	}

	return nil
}

type MockBlockStore struct {
	height       int64
	blockHistory map[int64]*tenderminttypes.Block
}

func (b *MockBlockStore) SetHeight(height int64) {
	b.height = height
}

func (b *MockBlockStore) AdvanceBlock(blockTime time.Duration) {
	// keep block height in mock blockstore
	blockInt64 := b.height + 1
	b.SetHeight(blockInt64)

	// create mock block header
	blockHeader := tenderminttypes.Header{}
	blockHeader.Height = blockInt64
	if prevBlock, ok := b.blockHistory[b.height-1]; ok {
		blockHeader.Time = prevBlock.Time.Add(blockTime)
	} else {
		blockHeader.Time = time.Now()
	}

	// update the blockstore's block history with current block
	b.SetBlockHistoryEntry(blockInt64, &tenderminttypes.Block{Header: blockHeader})
}

func (b *MockBlockStore) SetBlockHistoryEntry(height int64, blockCore *tenderminttypes.Block) {
	b.blockHistory[height] = blockCore
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
	return b.blockHistory[height]
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
