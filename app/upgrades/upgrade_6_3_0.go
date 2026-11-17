package upgrades

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/v5/app/keepers"
)

// VestingUnlockPauseDuration is the length of the v6.3.0 vesting unlock pause:
// 4 months of 30 days (120 days), in seconds, counted from the upgrade block time.
const VestingUnlockPauseDuration int64 = 120 * 24 * 60 * 60

// v_6_3_0 applies the vesting unlock pause to every periodic vesting account.
func v_6_3_0(
	m *module.Manager,
	c module.Configurator,
	_ BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		applyVestingUnlockPause(ctx, lk.AccountKeeper, VestingUnlockPauseDuration)
		return m.RunMigrations(ctx, c, vm)
	}
}

// applyVestingUnlockPause pauses the unlocking of every periodic vesting account for
// pauseDuration seconds, starting at the current block time. Tokens that already
// unlocked stay unlocked, and every tranche that has not unlocked yet moves
// pauseDuration seconds later. Other account types (including continuous and
// delayed vesting accounts) are not touched. It returns the number of paused accounts and
// the number of accounts skipped because they were invalid after the change.
func applyVestingUnlockPause(ctx sdk.Context, ak authkeeper.AccountKeeper, pauseDuration int64) (paused, skipped int) {
	blockTime := ctx.BlockTime().Unix()
	logger := ctx.Logger().With("upgrade", "vesting-unlock-pause")

	// GetAllAccounts collects all accounts first, so the store is not written while it is iterated
	for _, account := range ak.GetAllAccounts(ctx) {
		vestingAccount, ok := account.(*vestingtypes.PeriodicVestingAccount)
		if !ok {
			continue
		}

		oldStartTime, oldEndTime := vestingAccount.StartTime, vestingAccount.EndTime
		if !applyVestingUnlockPauseToAccount(vestingAccount, blockTime, pauseDuration) {
			continue
		}

		if err := vestingAccount.Validate(); err != nil {
			// not saved: the account keeps its original schedule
			skipped++
			logger.Error("vesting unlock pause skipped: account is invalid after the pause",
				"address", vestingAccount.Address, "error", err.Error())
			continue
		}

		ak.SetAccount(ctx, vestingAccount)
		paused++
		logger.Info("vesting unlock pause applied to account", "address", vestingAccount.Address,
			"old_start_time", oldStartTime, "new_start_time", vestingAccount.StartTime,
			"old_end_time", oldEndTime, "new_end_time", vestingAccount.EndTime)
	}

	logger.Info("vesting unlock pause applied", "paused_accounts", paused, "skipped_accounts", skipped,
		"pause_seconds", pauseDuration, "block_time", blockTime)
	return paused, skipped
}

// applyVestingUnlockPauseToAccount moves the part of the schedule that has not unlocked
// at blockTime pauseDuration seconds later, in place. It returns false, and changes
// nothing, when everything has already unlocked.
//
// It follows PeriodicVestingAccount.GetVestedCoins: nothing unlocks while
// blockTime <= StartTime, and a tranche unlocks once blockTime reaches StartTime
// plus the lengths of all tranches up to and including it.
func applyVestingUnlockPauseToAccount(account *vestingtypes.PeriodicVestingAccount, blockTime, pauseDuration int64) bool {
	if blockTime >= account.EndTime {
		return false // fully unlocked
	}

	if blockTime <= account.StartTime {
		// not started yet: move the whole schedule
		account.StartTime += pauseDuration
		account.EndTime += pauseDuration
		return true
	}

	// mid-schedule: lengthen the next tranche to unlock, which moves it and every later tranche
	unlockTime := account.StartTime
	for i := range account.VestingPeriods {
		unlockTime += account.VestingPeriods[i].Length
		if unlockTime > blockTime {
			account.VestingPeriods[i].Length += pauseDuration
			account.EndTime += pauseDuration
			return true
		}
	}

	return false // inconsistent schedule (tranches end before EndTime): leave it as it is
}
