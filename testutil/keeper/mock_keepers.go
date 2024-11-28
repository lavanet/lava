package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	tenderminttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// account keeper mock
type mockAccountKeeper struct{}

func (k mockAccountKeeper) IterateAccounts(ctx sdk.Context, process func(authtypes.AccountI) (stop bool)) {
}

func (k mockAccountKeeper) GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI {
	return nil
}

func (k mockAccountKeeper) GetModuleAccount(ctx sdk.Context, module string) authtypes.ModuleAccountI {
	moduleAddress := authtypes.NewModuleAddress(module).String()
	baseAccount := authtypes.NewBaseAccount(nil, nil, 0, 0)
	baseAccount.Address = moduleAddress
	return authtypes.NewModuleAccount(baseAccount, module, authtypes.Burner, authtypes.Staking)
}

func (k mockAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return GetModuleAddress(moduleName)
}

func (k mockAccountKeeper) SetModuleAccount(sdk.Context, authtypes.ModuleAccountI) {
}

// mock bank keeper
var balance map[string]sdk.Coins = make(map[string]sdk.Coins)

type mockStakingKeeperEmpty struct{}

func (k mockStakingKeeperEmpty) ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI {
	return nil
}

func (k mockStakingKeeperEmpty) UnbondingTime(ctx sdk.Context) time.Duration {
	return time.Duration(0)
}

func (k mockStakingKeeperEmpty) GetAllDelegatorDelegations(ctx sdk.Context, delegator sdk.AccAddress) []stakingtypes.Delegation {
	return nil
}

func (k mockStakingKeeperEmpty) GetDelegatorValidator(ctx sdk.Context, delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress) (validator stakingtypes.Validator, err error) {
	return stakingtypes.Validator{}, nil
}

func (k mockStakingKeeperEmpty) GetDelegation(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (delegation stakingtypes.Delegation, found bool) {
	return stakingtypes.Delegation{}, false
}

func (k mockStakingKeeperEmpty) GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool) {
	return stakingtypes.Validator{}, false
}

func (k mockStakingKeeperEmpty) GetValidatorDelegations(ctx sdk.Context, valAddr sdk.ValAddress) (delegations []stakingtypes.Delegation) {
	return []stakingtypes.Delegation{}
}

func (k mockStakingKeeperEmpty) BondDenom(ctx sdk.Context) string {
	return "ulava"
}

func (k mockStakingKeeperEmpty) ValidateUnbondAmount(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (shares sdk.Dec, err error) {
	return sdk.Dec{}, nil
}

func (k mockStakingKeeperEmpty) Undelegate(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount sdk.Dec) (time.Time, error) {
	return time.Time{}, nil
}

func (k mockStakingKeeperEmpty) Delegate(ctx sdk.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator, subtractAccount bool) (newShares sdk.Dec, err error) {
	return sdk.Dec{}, nil
}

func (k mockStakingKeeperEmpty) GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator {
	return []stakingtypes.Validator{}
}

func (k mockStakingKeeperEmpty) GetAllValidators(ctx sdk.Context) (validators []stakingtypes.Validator) {
	return []stakingtypes.Validator{}
}

type mockBankKeeper struct{}

func init_balance() {
	balance = make(map[string]sdk.Coins)
}

func (k mockBankKeeper) SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return nil
}

func (k mockBankKeeper) GetSupply(ctx sdk.Context, denom string) sdk.Coin {
	total := sdk.NewCoin(denom, sdk.ZeroInt())
	for _, coins := range balance {
		for _, coin := range coins {
			if coin.Denom == denom {
				total = total.Add(coin)
			}
		}
	}
	return total
}

func (k mockBankKeeper) LockedCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return k.GetAllBalances(ctx, addr)
}

func (k mockBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins := balance[addr.String()]
	for i := 0; i < coins.Len(); i++ {
		if coins.GetDenomByIndex(i) == denom {
			return coins[i]
		}
	}
	return sdk.NewCoin(denom, sdk.ZeroInt())
}

func (k mockBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return balance[addr.String()]
}

func (k mockBankKeeper) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	moduleAcc := GetModuleAddress(recipientModule)
	accountCoins := k.GetAllBalances(ctx, senderAddr)
	if !accountCoins.IsAllGTE(amt) {
		return fmt.Errorf("not enough coins")
	}

	k.SubFromBalance(senderAddr, amt)
	k.AddToBalance(moduleAcc, amt)
	return nil
}

func (k mockBankKeeper) UndelegateCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return k.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

func (k mockBankKeeper) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	moduleAcc := GetModuleAddress(senderModule)
	accountCoins := k.GetAllBalances(ctx, moduleAcc)
	if !accountCoins.IsAllGTE(amt) {
		return fmt.Errorf("not enough coins")
	}

	k.SubFromBalance(moduleAcc, amt)
	k.AddToBalance(recipientAddr, amt)
	return nil
}

func (k mockBankKeeper) SendCoinsFromModuleToModule(ctx sdk.Context, senderModule string, recipientModule string, amt sdk.Coins) error {
	senderModuleAcc := GetModuleAddress(senderModule)
	return k.SendCoinsFromAccountToModule(ctx, senderModuleAcc, recipientModule, amt)
}

func (k mockBankKeeper) DelegateCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	// TODO support multiple coins
	return k.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt)
}

func (k mockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	acc := GetModuleAddress(moduleName)
	k.AddToBalance(acc, amounts)
	return nil
}

func (k mockBankKeeper) BurnCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	acc := GetModuleAddress(moduleName)
	k.SubFromBalance(acc, amounts)
	return nil
}

func (k mockBankKeeper) SetBalance(ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins) error {
	balance[addr.String()] = amounts
	return nil
}

func (k mockBankKeeper) AddToBalance(addr sdk.AccAddress, amounts sdk.Coins) error {
	if _, ok := balance[addr.String()]; ok {
		for _, coin := range amounts {
			balance[addr.String()] = balance[addr.String()].Add(coin)
		}
	} else {
		balance[addr.String()] = amounts
	}
	return nil
}

func (k mockBankKeeper) SubFromBalance(addr sdk.AccAddress, amounts sdk.Coins) error {
	if _, ok := balance[addr.String()]; ok {
		balance[addr.String()] = balance[addr.String()].Sub(amounts...)
	} else {
		return fmt.Errorf("acount is empty, can't sub")
	}

	return nil
}

func (k mockBankKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	return false
}

type MockBlockStore struct {
	height       int64
	blockHistory map[int64]*tenderminttypes.Block
}

var (
	fixedTime bool
	fixedDate = time.Date(2024, time.March, 1, 1, 1, 1, 1, time.UTC)
)

func (b *MockBlockStore) SetHeight(height int64) {
	b.height = height
}

func SetFixedTime() {
	fixedTime = true
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
	} else if !fixedTime {
		blockHeader.Time = time.Now().Add(blockTime)
	} else {
		blockHeader.Time = fixedDate.Add(blockTime)
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

func (b *MockBlockStore) LoadBlockMetaByHash(hash []byte) *tenderminttypes.BlockMeta {
	return nil
}

func (b *MockBlockStore) DeleteLatestBlock() error {
	return nil
}
