package keeper

import (
	"fmt"

	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/lavanet/lava/x/rewards/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		accountKeeper      types.AccountKeeper
		specKeeper         types.SpecKeeper
		epochstorage       types.EpochstorageKeeper
		downtimeKeeper     types.DowntimeKeeper
		stakingKeeper      types.StakingKeeper
		dualstakingKeeper  types.DualStakingKeeper
		distributionKeeper types.DistributionKeeper

		// account name used by the distribution module to reward validators
		feeCollectorName string

		// used to operate the monthly refill of the validators and providers rewards pool mechanism
		// there is always a single timer that is expired in the next month
		// the timer subkey holds the block in which the timer will expire (not exact)
		// the timer data holds the number of months left for the allocation pools (until all funds are gone)
		refillRewardsPoolTS timerstoretypes.TimerStore

		// the address capable of executing a MsgSetIprpcData message. Typically, this
		// should be the x/gov module account.
		authority string

		// ICS4 wrapper that lets the rewards module be an IBC middleware module
		ics4Wrapper porttypes.ICS4Wrapper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	specKeeper types.SpecKeeper,
	epochStorageKeeper types.EpochstorageKeeper,
	downtimeKeeper types.DowntimeKeeper,
	stakingKeeper types.StakingKeeper,
	dualstakingKeeper types.DualStakingKeeper,
	distributionKeeper types.DistributionKeeper,
	feeCollectorName string,
	timerStoreKeeper types.TimerStoreKeeper,
	authority string,
	ics4Wrapper porttypes.ICS4Wrapper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	keeper := Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,

		bankKeeper:         bankKeeper,
		accountKeeper:      accountKeeper,
		specKeeper:         specKeeper,
		epochstorage:       epochStorageKeeper,
		downtimeKeeper:     downtimeKeeper,
		stakingKeeper:      stakingKeeper,
		dualstakingKeeper:  dualstakingKeeper,
		distributionKeeper: distributionKeeper,

		feeCollectorName: feeCollectorName,
		authority:        authority,
		ics4Wrapper:      ics4Wrapper,
	}

	refillRewardsPoolTimerCallback := func(ctx sdk.Context, subkey, data []byte) {
		keeper.distributeMonthlyBonusRewards(ctx)
		keeper.RefillRewardsPools(ctx, subkey, data)
	}

	// making an EndBlock timer store to make sure it'll happen after the BeginBlock that pays validators
	keeper.refillRewardsPoolTS = *timerStoreKeeper.NewTimerStoreEndBlock(storeKey, types.RefillRewardsPoolTimerPrefix).
		WithCallbackByBlockTime(refillRewardsPoolTimerCallback)

	return &keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// redeclaring BeginBlock for testing (this is not called outside of unit tests)
func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.DistributeBlockReward(ctx)
	k.RemoveExpiredPendingIbcIprpcFunds(ctx)
}

func (k Keeper) GetModuleAddress() string {
	return k.accountKeeper.GetModuleAddress(types.ModuleName).String()
}

// BondedTargetFactor calculates the bonded target factor which is used to calculate the validators
// block rewards
func (k Keeper) BondedTargetFactor(ctx sdk.Context) cosmosMath.LegacyDec {
	params := k.GetParams(ctx)

	minBonded := params.MinBondedTarget
	maxBonded := params.MaxBondedTarget
	lowFactor := params.LowFactor
	bonded := k.stakingKeeper.BondedRatio(ctx)

	if bonded.GT(maxBonded) {
		return lowFactor
	}

	if bonded.LTE(minBonded) {
		return cosmosMath.LegacyOneDec()
	} else {
		// equivalent to: (maxBonded - bonded) / (maxBonded - minBonded)
		// 					  + lowFactor * (bonded - minBonded) / (maxBonded - minBonded)
		min_max_diff := maxBonded.Sub(minBonded)

		e1 := maxBonded.Sub(bonded).Quo(min_max_diff)
		e2 := bonded.Sub(minBonded).Quo(min_max_diff)
		return e1.Add(e2.Mul(lowFactor))
	}
}

// InitRewardsRefillTS initializes the refill pools' timer store
func (k Keeper) InitRewardsRefillTS(ctx sdk.Context, gs timerstoretypes.GenesisState) {
	k.refillRewardsPoolTS.Init(ctx, gs)
}

// ExportRewardsRefillTS exports refill pools timers data (for genesis)
func (k Keeper) ExportRewardsRefillTS(ctx sdk.Context) timerstoretypes.GenesisState {
	return k.refillRewardsPoolTS.Export(ctx)
}

// ICS4 wrapper default implementations
func (k Keeper) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, sourcePort string, sourceChannel string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte,
) (sequence uint64, err error) {
	return k.ics4Wrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (k Keeper) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	return k.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

func (k Keeper) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return k.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}
