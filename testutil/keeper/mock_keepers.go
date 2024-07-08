package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	tenderminttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
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

// Mock IBC transfer keeper
type mockIbcTransferKeeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramstypes.Subspace
	authKeeper mockAccountKeeper
	bankKeeper mockBankKeeper
}

const (
	MockSrcPort     = "src"
	MockSrcChannel  = "srcc"
	MockDestPort    = "dest"
	MockDestChannel = "destc"
)

func NewMockIbcTransferKeeper(storeKey storetypes.StoreKey, cdc codec.BinaryCodec, paramSpace paramstypes.Subspace, authKeeper mockAccountKeeper, bankKeeper mockBankKeeper) mockIbcTransferKeeper {
	return mockIbcTransferKeeper{
		storeKey:   storeKey,
		cdc:        cdc,
		paramSpace: paramSpace,
		authKeeper: authKeeper,
		bankKeeper: bankKeeper,
	}
}

func (k mockIbcTransferKeeper) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, channelCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string) (string, error) {
	return "", nil
}

func (k mockIbcTransferKeeper) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, channelCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, counterpartyVersion string) (version string, err error) {
	return "", nil
}

func (k mockIbcTransferKeeper) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string) error {
	return nil
}

func (k mockIbcTransferKeeper) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return nil
}

func (k mockIbcTransferKeeper) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return nil
}

func (k mockIbcTransferKeeper) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return nil
}

func (k mockIbcTransferKeeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	// get data from packet
	var data ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// parse data
	senderAcc, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	receiverAcc, err := sdk.AccAddressFromBech32(data.Receiver)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	amount, ok := math.NewIntFromString(data.Amount)
	if !ok {
		return channeltypes.NewErrorAcknowledgement(fmt.Errorf("invalid amount in transfer data: %s", data.Amount))
	}

	// sub balance from sender in original denom, add balance to receiver (on other chain) with IBC Denom
	coins := sdk.NewCoins(sdk.NewCoin(data.Denom, amount))
	err = k.bankKeeper.SubFromBalance(senderAcc, coins)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	ibcCoins := sdk.NewCoins(ibctransfertypes.GetTransferCoin(packet.DestinationPort, packet.DestinationChannel, data.Denom, amount))
	err = k.bankKeeper.AddToBalance(receiverAcc, ibcCoins)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement([]byte{byte(1)})
}

func (k mockIbcTransferKeeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	return nil
}

func (k mockIbcTransferKeeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	return nil
}

// Mock BlockStore
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
