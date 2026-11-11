package upgrades

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/v5/app/keepers"
	"github.com/stretchr/testify/require"
)

const (
	day          = int64(24 * 60 * 60)
	dailyTranche = int64(18_264_840_182) // ulava, the daily tranche of a 20M LAVA mainnet account
)

// mainnetVestingStart is the start of the mainnet genesis vesting schedules (after the v4.2.0 shift)
var mainnetVestingStart = time.Date(2026, 1, 12, 15, 0, 0, 0, time.UTC).Unix()

func testAddress(name string) sdk.AccAddress {
	addr := make([]byte, 20)
	copy(addr, name)
	return addr
}

// newMainnetShapedVestingAccount mirrors a mainnet genesis vesting account: a cliff
// tranche of 1/3 that unlocks at StartTime, then 730 equal daily tranches, with part
// of the locked tokens delegated.
func newMainnetShapedVestingAccount(addr sdk.AccAddress, startTime int64) *vestingtypes.PeriodicVestingAccount {
	cliff := sdk.NewCoins(sdk.NewInt64Coin("ulava", 6_666_666_667_140))
	daily := sdk.NewCoins(sdk.NewInt64Coin("ulava", dailyTranche))
	periods := vestingtypes.Periods{{Length: 0, Amount: cliff}}
	total := cliff
	for i := 0; i < 730; i++ {
		periods = append(periods, vestingtypes.Period{Length: day, Amount: daily})
		total = total.Add(daily...)
	}
	account := vestingtypes.NewPeriodicVestingAccount(authtypes.NewBaseAccountWithAddress(addr), total, startTime, periods)
	account.DelegatedVesting = sdk.NewCoins(sdk.NewInt64Coin("ulava", 14_336_792_801_522))
	return account
}

func vestedAt(account *vestingtypes.PeriodicVestingAccount, unix int64) sdk.Coins {
	return account.GetVestedCoins(time.Unix(unix, 0))
}

func TestApplyVestingUnlockPauseToAccount(t *testing.T) {
	const pause = VestingUnlockPauseDuration
	start := mainnetVestingStart
	end := start + 730*day

	tests := []struct {
		name          string
		blockTime     int64
		expectChanged bool
	}{
		{name: "before start", blockTime: start - 30*day, expectChanged: true},
		{name: "at start, cliff still locked", blockTime: start, expectChanged: true},
		{name: "right after the cliff unlocked", blockTime: start + 1, expectChanged: true},
		{name: "between two daily unlocks", blockTime: start + 322*day - 5*60*60, expectChanged: true},
		{name: "one second before a daily unlock", blockTime: start + 100*day - 1, expectChanged: true},
		{name: "exactly at a daily unlock", blockTime: start + 100*day, expectChanged: true},
		{name: "one second before the last unlock", blockTime: end - 1, expectChanged: true},
		{name: "at the last unlock", blockTime: end, expectChanged: false},
		{name: "after the last unlock", blockTime: end + day, expectChanged: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := testAddress("vesting_account")
			original := newMainnetShapedVestingAccount(addr, start)
			paused := newMainnetShapedVestingAccount(addr, start)
			now := tt.blockTime

			require.Equal(t, tt.expectChanged, applyVestingUnlockPauseToAccount(paused, now, pause))
			if !tt.expectChanged {
				require.Equal(t, original, paused)
				return
			}

			// the schedule stays valid and only its timing changes
			require.NoError(t, paused.Validate())
			require.Equal(t, original.EndTime+pause, paused.EndTime)
			require.Equal(t, original.OriginalVesting, paused.OriginalVesting)
			require.Equal(t, original.DelegatedVesting, paused.DelegatedVesting)
			require.Equal(t, original.DelegatedFree, paused.DelegatedFree)
			require.Len(t, paused.VestingPeriods, len(original.VestingPeriods))
			lengthened := 0
			for i := range original.VestingPeriods {
				require.Equal(t, original.VestingPeriods[i].Amount, paused.VestingPeriods[i].Amount)
				if diff := paused.VestingPeriods[i].Length - original.VestingPeriods[i].Length; diff != 0 {
					require.Equal(t, pause, diff)
					lengthened++
				}
			}
			if now <= start {
				require.Equal(t, original.StartTime+pause, paused.StartTime)
				require.Zero(t, lengthened)
			} else {
				require.Equal(t, original.StartTime, paused.StartTime)
				require.Equal(t, 1, lengthened)
			}

			// what unlocked until now stays unlocked
			for _, past := range []int64{now - 2*day, now - 1, now} {
				require.Equal(t, vestedAt(original, past), vestedAt(paused, past), "at now%+d", past-now)
			}

			// nothing unlocks during the pause
			unlockedAtPause := vestedAt(original, now)
			for _, during := range []int64{now + 1, now + day, now + pause/2, now + pause - 1} {
				require.Equal(t, unlockedAtPause, vestedAt(paused, during), "at now%+d", during-now)
			}

			// after the pause, unlocking continues from the same point, shifted by the pause
			for _, after := range []int64{0, 1, day - 1, day, 37*day + 123, end - now, end - now + day} {
				require.Equal(t, vestedAt(original, now+after), vestedAt(paused, now+pause+after), "at now+pause%+d", after)
			}
		})
	}
}

func TestApplyVestingUnlockPauseToAccount_MainnetDates(t *testing.T) {
	account := newMainnetShapedVestingAccount(testAddress("vesting_account"), mainnetVestingStart)
	upgradeTime := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC) // next daily unlock: 2026-12-01 15:00

	require.True(t, applyVestingUnlockPauseToAccount(account, upgradeTime.Unix(), VestingUnlockPauseDuration))

	unlockedAtUpgrade := account.GetVestedCoins(upgradeTime)
	resume := time.Date(2027, 3, 31, 15, 0, 0, 0, time.UTC) // 2026-12-01 15:00 + 120 days
	require.Equal(t, unlockedAtUpgrade, account.GetVestedCoins(resume.Add(-time.Second)))
	require.Equal(t, unlockedAtUpgrade.Add(sdk.NewInt64Coin("ulava", dailyTranche)), account.GetVestedCoins(resume))
	require.Equal(t, time.Date(2028, 5, 11, 15, 0, 0, 0, time.UTC).Unix(), account.EndTime) // 2028-01-12 15:00 + 120 days
}

func TestApplyVestingUnlockPauseToAccount_InconsistentSchedule(t *testing.T) {
	newInconsistentAccount := func() *vestingtypes.PeriodicVestingAccount {
		account := newMainnetShapedVestingAccount(testAddress("vesting_account"), mainnetVestingStart)
		account.EndTime += 10 * day // tranches now end 10 days before EndTime
		return account
	}
	account, original := newInconsistentAccount(), newInconsistentAccount()

	blockTime := mainnetVestingStart + 735*day // after the last tranche, before EndTime
	require.False(t, applyVestingUnlockPauseToAccount(account, blockTime, VestingUnlockPauseDuration))
	require.Equal(t, original, account)
}

func newTestAccountKeeper(t *testing.T, blockTime time.Time) (sdk.Context, authkeeper.AccountKeeper, moduletestutil.TestEncodingConfig) {
	key := storetypes.NewKVStoreKey(authtypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, vesting.AppModuleBasic{})
	ak := authkeeper.NewAccountKeeper(encCfg.Codec, key, authtypes.ProtoBaseAccount, map[string][]string{},
		"lava@", authtypes.NewModuleAddress(govtypes.ModuleName).String())
	return testCtx.Ctx.WithBlockTime(blockTime), ak, encCfg
}

func TestApplyVestingUnlockPause_OnlyPeriodicAccountsChange(t *testing.T) {
	upgradeTime := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)
	now := upgradeTime.Unix()
	ctx, ak, _ := newTestAccountKeeper(t, upgradeTime)
	coins := sdk.NewCoins(sdk.NewInt64Coin("ulava", 1_000_000))

	accounts := map[string]authtypes.AccountI{
		"base":              authtypes.NewBaseAccountWithAddress(testAddress("base")),
		"periodic mid":      newMainnetShapedVestingAccount(testAddress("periodic_mid"), mainnetVestingStart),
		"periodic future":   newMainnetShapedVestingAccount(testAddress("periodic_future"), now+30*day),
		"periodic finished": newMainnetShapedVestingAccount(testAddress("periodic_finished"), now-800*day),
		"continuous": vestingtypes.NewContinuousVestingAccount(
			authtypes.NewBaseAccountWithAddress(testAddress("continuous")), coins, now-100*day, now+100*day),
		"delayed": vestingtypes.NewDelayedVestingAccount(
			authtypes.NewBaseAccountWithAddress(testAddress("delayed")), coins, now+100*day),
	}
	expectPaused := map[string]bool{"periodic mid": true, "periodic future": true}

	before := map[string]authtypes.AccountI{}
	for name, account := range accounts {
		ak.SetAccount(ctx, ak.NewAccount(ctx, account))
		before[name] = ak.GetAccount(ctx, account.GetAddress())
	}

	require.Equal(t, len(expectPaused), applyVestingUnlockPause(ctx, ak, VestingUnlockPauseDuration))

	for name, account := range accounts {
		after := ak.GetAccount(ctx, account.GetAddress())
		if !expectPaused[name] {
			require.Equal(t, before[name], after, name)
			continue
		}
		expected, ok := before[name].(*vestingtypes.PeriodicVestingAccount)
		require.True(t, ok, name)
		require.True(t, applyVestingUnlockPauseToAccount(expected, now, VestingUnlockPauseDuration), name)
		require.Equal(t, expected, after, name)
	}
}

func TestUpgrade_6_3_0_AppliesVestingUnlockPause(t *testing.T) {
	upgradeTime := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)
	ctx, ak, encCfg := newTestAccountKeeper(t, upgradeTime)
	account := newMainnetShapedVestingAccount(testAddress("periodic_mid"), mainnetVestingStart)
	ak.SetAccount(ctx, ak.NewAccount(ctx, account))

	configurator := module.NewConfigurator(encCfg.Codec, baseapp.NewMsgServiceRouter(), baseapp.NewGRPCQueryRouter())
	handler := Upgrade_6_3_0.CreateUpgradeHandler(module.NewManager(), configurator, nil, &keepers.LavaKeepers{AccountKeeper: ak})
	_, err := handler(ctx, upgradetypes.Plan{Name: Upgrade_6_3_0.UpgradeName}, module.VersionMap{})
	require.NoError(t, err)

	after, ok := ak.GetAccount(ctx, account.GetAddress()).(*vestingtypes.PeriodicVestingAccount)
	require.True(t, ok)
	require.Equal(t, account.EndTime+VestingUnlockPauseDuration, after.EndTime)
	require.Equal(t, account.GetVestedCoins(upgradeTime), after.GetVestedCoins(upgradeTime.Add(119*24*time.Hour)))
}
