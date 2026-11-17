package upgrades

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/v5/app/keepers"
)

// VestingUnlockPauseDuration is the length of the v6.3.0 vesting unlock pause:
// 4 months of 30 days (120 days), in seconds, counted from the upgrade block time.
const VestingUnlockPauseDuration int64 = 120 * 24 * 60 * 60

// vestingUnlockPauseTrancheLength is the length of the tranches a paused continuous vesting account
// unlocks in: one day, the cadence of the genesis periodic vesting accounts.
const vestingUnlockPauseTrancheLength int64 = 24 * 60 * 60

// pausedVestingAccount is a vesting account the pause saves after validating it
type pausedVestingAccount interface {
	vestingexported.VestingAccount
	Validate() error
}

// v_6_3_0 applies the vesting unlock pause to every periodic and continuous vesting account.
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

// applyVestingUnlockPause pauses the unlocking of every periodic and continuous vesting account
// for pauseDuration seconds, starting at the current block time. Tokens that already unlocked stay
// unlocked, and everything that has not unlocked yet unlocks pauseDuration seconds later. Delayed
// vesting accounts and other account types are not touched. It returns the number of paused
// accounts and the number of unfinished accounts skipped because the pause could not be applied.
func applyVestingUnlockPause(ctx sdk.Context, ak authkeeper.AccountKeeper, pauseDuration int64) (paused, skipped int) {
	blockTime := ctx.BlockTime().Unix()
	logger := ctx.Logger().With("upgrade", "vesting-unlock-pause")

	// GetAllAccounts collects all accounts first, so the store is not written while it is iterated
	for _, account := range ak.GetAllAccounts(ctx) {
		vestingAccount, ok := account.(vestingexported.VestingAccount)
		if !ok {
			continue
		}
		oldStartTime, oldEndTime := vestingAccount.GetStartTime(), vestingAccount.GetEndTime()

		var pausedAccount pausedVestingAccount
		convertedToPeriodic := false
		switch typedAccount := account.(type) {
		case *vestingtypes.PeriodicVestingAccount:
			if !applyVestingUnlockPauseToAccount(typedAccount, blockTime, pauseDuration) {
				if blockTime < typedAccount.EndTime {
					// not fully unlocked, but its tranches end before EndTime: it misses the pause
					skipped++
					logger.Error("vesting unlock pause skipped: inconsistent vesting schedule",
						"address", typedAccount.Address)
				}
				continue
			}
			pausedAccount = typedAccount
		case *vestingtypes.ContinuousVestingAccount:
			converted, changed := applyVestingUnlockPauseToContinuousAccount(typedAccount, blockTime, pauseDuration)
			if !changed {
				continue
			}
			pausedAccount = converted
			_, convertedToPeriodic = converted.(*vestingtypes.PeriodicVestingAccount)
		default:
			continue
		}

		if err := pausedAccount.Validate(); err != nil {
			// not saved: the account keeps its original schedule
			skipped++
			logger.Error("vesting unlock pause skipped: account is invalid after the pause",
				"address", pausedAccount.GetAddress().String(), "error", err.Error())
			continue
		}

		ak.SetAccount(ctx, pausedAccount)
		paused++
		logger.Info("vesting unlock pause applied to account", "address", pausedAccount.GetAddress().String(),
			"converted_to_periodic", convertedToPeriodic, "old_start_time", oldStartTime, "new_start_time", pausedAccount.GetStartTime(),
			"old_end_time", oldEndTime, "new_end_time", pausedAccount.GetEndTime())
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

// applyVestingUnlockPauseToContinuousAccount pauses a continuous vesting account. It returns false,
// and changes nothing, when everything has already unlocked.
//
// An account that has not started keeps its type, and its start and end times move pauseDuration
// seconds later. A continuous schedule can't hold a gap, so an account that is mid-schedule becomes a
// periodic one with the same base account (balances, delegations, original vesting): a first tranche of
// everything vested at blockTime, which unlocks at blockTime as before, then the rest in daily tranches
// from the end of the pause to EndTime + pauseDuration, in proportion to time like the original schedule.
func applyVestingUnlockPauseToContinuousAccount(account *vestingtypes.ContinuousVestingAccount, blockTime, pauseDuration int64,
) (pausedVestingAccount, bool) {
	if blockTime >= account.EndTime {
		return nil, false // fully unlocked
	}

	if blockTime <= account.StartTime {
		// not started yet: nothing has unlocked, so moving the whole schedule re-locks nothing
		account.StartTime += pauseDuration
		account.EndTime += pauseDuration
		return account, true
	}

	vested := sdk.NewCoins(account.GetVestedCoins(time.Unix(blockTime, 0))...)
	remaining := account.OriginalVesting.Sub(vested...)
	periods := vestingtypes.Periods{{Length: blockTime - account.StartTime, Amount: vested}}

	resumeTime, newEndTime := blockTime+pauseDuration, account.EndTime+pauseDuration
	remainingDuration := math.NewInt(newEndTime - resumeTime)
	previousUnlockTime, previousUnlocked := blockTime, sdk.NewCoins()
	for unlockTime := resumeTime + vestingUnlockPauseTrancheLength; previousUnlockTime < newEndTime; unlockTime += vestingUnlockPauseTrancheLength {
		unlockTime = min(unlockTime, newEndTime)
		unlocked := sdk.NewCoins() // unlocked by unlockTime, out of the remaining coins
		for _, coin := range remaining {
			amount := coin.Amount.Mul(math.NewInt(unlockTime - resumeTime)).Quo(remainingDuration)
			unlocked = unlocked.Add(sdk.NewCoin(coin.Denom, amount))
		}
		periods = append(periods, vestingtypes.Period{Length: unlockTime - previousUnlockTime, Amount: unlocked.Sub(previousUnlocked...)})
		previousUnlockTime, previousUnlocked = unlockTime, unlocked
	}

	account.BaseVestingAccount.EndTime = newEndTime
	return vestingtypes.NewPeriodicVestingAccountRaw(account.BaseVestingAccount, account.StartTime, periods), true
}
