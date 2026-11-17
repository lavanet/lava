package upgrades

import (
	"encoding/json"
	"os"
	"strconv"
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
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
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

// newMonthlyVestingAccount mirrors the monthly periodic accounts created on mainnet after
// genesis: a 1-second first tranche, then 24 tranches of 30 days.
func newMonthlyVestingAccount(addr sdk.AccAddress, startTime int64) *vestingtypes.PeriodicVestingAccount {
	first := sdk.NewCoins(sdk.NewInt64Coin("ulava", 1_000_000_000))
	monthly := sdk.NewCoins(sdk.NewInt64Coin("ulava", 50_000_000_000))
	periods := vestingtypes.Periods{{Length: 1, Amount: first}}
	total := first
	for i := 0; i < 24; i++ {
		periods = append(periods, vestingtypes.Period{Length: 30 * day, Amount: monthly})
		total = total.Add(monthly...)
	}
	return vestingtypes.NewPeriodicVestingAccount(authtypes.NewBaseAccountWithAddress(addr), total, startTime, periods)
}

// unlockTimes returns the time each tranche of the account unlocks
func unlockTimes(account *vestingtypes.PeriodicVestingAccount) []int64 {
	times := make([]int64, 0, len(account.VestingPeriods))
	unlockTime := account.StartTime
	for _, period := range account.VestingPeriods {
		unlockTime += period.Length
		times = append(times, unlockTime)
	}
	return times
}

func vestedAt(account *vestingtypes.PeriodicVestingAccount, unix int64) sdk.Coins {
	return account.GetVestedCoins(time.Unix(unix, 0))
}

func TestApplyVestingUnlockPauseToAccount(t *testing.T) {
	start := mainnetVestingStart
	shapes := []struct {
		name       string
		newAccount func(sdk.AccAddress, int64) *vestingtypes.PeriodicVestingAccount
	}{
		{name: "genesis shape", newAccount: newMainnetShapedVestingAccount},
		{name: "monthly shape", newAccount: newMonthlyVestingAccount},
	}

	for _, shape := range shapes {
		unlocks := unlockTimes(shape.newAccount(testAddress("vesting_account"), start))
		end := unlocks[len(unlocks)-1]
		middle := unlocks[len(unlocks)/2]
		tests := []struct {
			name          string
			blockTime     int64
			expectChanged bool
		}{
			{name: "before start", blockTime: start - 30*day, expectChanged: true},
			{name: "at start, nothing unlocked yet", blockTime: start, expectChanged: true},
			{name: "right after the first unlock", blockTime: unlocks[0] + 1, expectChanged: true},
			{name: "between two unlocks", blockTime: middle - 5*60*60, expectChanged: true},
			{name: "one second before an unlock", blockTime: middle - 1, expectChanged: true},
			{name: "exactly at an unlock", blockTime: middle, expectChanged: true},
			{name: "one second before the last unlock", blockTime: end - 1, expectChanged: true},
			{name: "at the last unlock", blockTime: end, expectChanged: false},
			{name: "after the last unlock", blockTime: end + day, expectChanged: false},
		}

		for _, tt := range tests {
			t.Run(shape.name+"/"+tt.name, func(t *testing.T) {
				addr := testAddress("vesting_account")
				original := shape.newAccount(addr, start)
				paused := shape.newAccount(addr, start)
				requirePauseAt(t, original, paused, tt.blockTime, tt.expectChanged)
			})
		}
	}
}

// requirePauseAt applies the pause at now to paused (a copy of original) and checks the
// result against the SDK's own GetVestedCoins.
func requirePauseAt(t *testing.T, original, paused *vestingtypes.PeriodicVestingAccount, now int64, expectChanged bool) {
	const pause = VestingUnlockPauseDuration
	start, end := original.StartTime, original.EndTime

	require.Equal(t, expectChanged, applyVestingUnlockPauseToAccount(paused, now, pause))
	if !expectChanged {
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
	// ica registers the interchain account type, which also exists on mainnet
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, vesting.AppModuleBasic{}, ica.AppModuleBasic{})
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

	paused, skipped := applyVestingUnlockPause(ctx, ak, VestingUnlockPauseDuration)
	require.Equal(t, len(expectPaused), paused)
	require.Zero(t, skipped)

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

func TestApplyVestingUnlockPause_SkipsInvalidAccount(t *testing.T) {
	upgradeTime := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		corrupt func(account *vestingtypes.PeriodicVestingAccount)
	}{
		{
			name: "fails validation after the change",
			corrupt: func(account *vestingtypes.PeriodicVestingAccount) {
				account.DelegatedVesting = account.OriginalVesting.Add(sdk.NewInt64Coin("ulava", 1))
			},
		},
		{
			name: "inconsistent schedule that has not fully unlocked",
			corrupt: func(account *vestingtypes.PeriodicVestingAccount) {
				// the tranches end before the upgrade, but EndTime is after it
				account.StartTime = upgradeTime.Unix() - 800*day
				account.EndTime = upgradeTime.Unix() + 10*day
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, ak, _ := newTestAccountKeeper(t, upgradeTime)
			valid := newMainnetShapedVestingAccount(testAddress("periodic_valid"), mainnetVestingStart)
			invalid := newMainnetShapedVestingAccount(testAddress("periodic_invalid"), mainnetVestingStart)
			tt.corrupt(invalid)
			ak.SetAccount(ctx, ak.NewAccount(ctx, valid))
			ak.SetAccount(ctx, ak.NewAccount(ctx, invalid))
			invalidBefore := ak.GetAccount(ctx, invalid.GetAddress())

			paused, skipped := applyVestingUnlockPause(ctx, ak, VestingUnlockPauseDuration)
			require.Equal(t, 1, paused)
			require.Equal(t, 1, skipped)
			require.Equal(t, invalidBefore, ak.GetAccount(ctx, invalid.GetAddress()), "a skipped account keeps its original schedule")
		})
	}
}

// TestApplyVestingUnlockPause_StateExport is a dry run of the vesting unlock pause against a state
// export, to run before the upgrade proposal. It is skipped unless LAVA_STATE_EXPORT is set:
//
//	lavad export > export.json
//	LAVA_STATE_EXPORT=export.json VESTING_UNLOCK_PAUSE_TIME=2026-12-01T10:00:00Z \
//	  VESTING_UNLOCK_PAUSE_EXPECTED_PAUSED=229 \
//	  go test ./app/upgrades -run TestApplyVestingUnlockPause_StateExport -v
//
// LAVA_STATE_EXPORT is a full export or just its auth section (jq '.app_state.auth' export.json).
// VESTING_UNLOCK_PAUSE_TIME is the expected upgrade block time (RFC 3339) and defaults to now.
// VESTING_UNLOCK_PAUSE_EXPECTED_PAUSED, if set, is the number of accounts expected to be paused.
// The test fails if any account would be skipped or the paused count differs from the expected one.
func TestApplyVestingUnlockPause_StateExport(t *testing.T) {
	exportPath := os.Getenv("LAVA_STATE_EXPORT")
	if exportPath == "" {
		t.Skip("set LAVA_STATE_EXPORT to a state export (lavad export) to run this dry run")
	}
	pauseTime := time.Now().UTC()
	if value := os.Getenv("VESTING_UNLOCK_PAUSE_TIME"); value != "" {
		var err error
		pauseTime, err = time.Parse(time.RFC3339, value)
		require.NoError(t, err)
	}
	ctx, ak, encCfg := newTestAccountKeeper(t, pauseTime)

	// exported addresses use the lava@ prefix
	sdkConfig := sdk.GetConfig()
	accountPrefix, accountPubPrefix := sdkConfig.GetBech32AccountAddrPrefix(), sdkConfig.GetBech32AccountPubPrefix()
	sdkConfig.SetBech32PrefixForAccount("lava@", "lava@pub")
	t.Cleanup(func() { sdkConfig.SetBech32PrefixForAccount(accountPrefix, accountPubPrefix) })

	exportJSON, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	var export struct {
		AppState map[string]json.RawMessage `json:"app_state"`
	}
	require.NoError(t, json.Unmarshal(exportJSON, &export))
	authJSON := exportJSON
	if export.AppState != nil {
		authJSON = export.AppState[authtypes.ModuleName]
	}
	var authGenesis authtypes.GenesisState
	require.NoError(t, encCfg.Codec.UnmarshalJSON(authJSON, &authGenesis))
	accounts, err := authtypes.UnpackAccounts(authGenesis.Accounts)
	require.NoError(t, err)

	periodic := 0
	for _, account := range accounts {
		if _, ok := account.(*vestingtypes.PeriodicVestingAccount); ok {
			periodic++
		}
		ak.SetAccount(ctx, account)
	}

	paused, skipped := applyVestingUnlockPause(ctx, ak, VestingUnlockPauseDuration)
	t.Logf("pause time %s: %d accounts, %d periodic vesting, %d paused, %d skipped",
		pauseTime.Format(time.RFC3339), len(accounts), periodic, paused, skipped)
	require.Zero(t, skipped, "accounts that would miss the vesting unlock pause")
	if value := os.Getenv("VESTING_UNLOCK_PAUSE_EXPECTED_PAUSED"); value != "" {
		expectedPaused, err := strconv.Atoi(value)
		require.NoError(t, err)
		require.Equal(t, expectedPaused, paused, "paused accounts")
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
