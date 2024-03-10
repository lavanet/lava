package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Pool string

// Allocation pools constants:
// These pools are used as the main and reserve of tokens for the distribution rewards pools.
// Each month, the allocation pools transfer a monthly quota of tokens to the distribution
// pools so there will be funds to distribute rewards for validators/providers.
// The allocation pools will get depleted after RewardsAllocationPoolsLifetime.
const (
	ValidatorsRewardsAllocationPoolName Pool   = "validators_rewards_allocation_pool"
	ProvidersRewardsAllocationPool      Pool   = "providers_rewards_allocation_pool"
	RewardsAllocationPoolsLifetime      uint64 = 48 // 4 years (in months)
)

// Distribution pools constants:
// These pools are used as the reserve of tokens for validators/providers rewards.
// Once a month, these pools' tokens are burned by the burn rate and get their
// monthly quota of tokens from the allocation pools
const (
	ValidatorsRewardsDistributionPoolName Pool = "validators_rewards_distribution_pool"
	ProviderRewardsDistributionPool       Pool = "providers_rewards_distribution_pool"
	DistributionPoolRefillEventName            = "distribution_pools_refill"
	ProvidersBonusRewardsEventName             = "provider_bonus_rewards"
)

// BlocksToTimerExpirySlackFactor is used to calculate the number of blocks until the
// next timer expiry which determine the validators block rewards.
// since the time/blocks conversion can be errornous, we multiply our calculated number
// of blocks by this error margin, so we'll won't have a case of having too few blocks
var BlocksToTimerExpirySlackFactor math.LegacyDec = sdk.NewDecWithPrec(105, 2) // 1.05

// Refill reward pools time stores constants:
// This timer store is used to trigger the refill mechanism of the distribution
// pools once a month.
const (
	RefillRewardsPoolTimerPrefix = "refill-rewards-pool-ts"
	RefillRewardsPoolTimerName   = "refill-rewards-timer"
)

// IPRPC Pool:
// IPRPC (Incentivized Providers RPC) pool is meant to hold bonus rewards for providers that
// provide service on specific specs. The rewards from this pool are distributed on a monthly
// basis
const (
	IprpcPoolName                           Pool   = "iprpc_pool"
	IprpcPoolEmissionEventName              string = "iprpc-pool-emmission"
	SetIprpcDataEventName                          = "set-iprpc-data"
	FundIprpcEventName                             = "fund-iprpc"
	TransferIprpcRewardToNextMonthEventName        = "transfer-iprpc-reward-to-next-month"
)

// helper struct to track the serviced IPRPC CU for each spec+provider
type SpecCuType struct {
	ProvidersCu []ProviderCuType
	TotalCu     uint64
}

type ProviderCuType struct {
	Provider string
	CU       uint64
}
