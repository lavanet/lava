package types

type Pool string

// validators_rewards_pool constants
// this pool is used as the main and reserve of token for validators rewards
// it will get depleted after ValidatorsRewardsPoolLifetime (4 years minus one month)
const (
	ValidatorsRewardsPoolName     Pool  = "validators_rewards_pool"
	ValidatorsRewardsPoolLifetime int64 = 47
)

// validators block rewards pool constants
// this pool is used as the reserve for validator rewards per block
// it gets its token through the validators rewards pool each month
// this monthly transfer happens using the "refill reward pools" timer store
const (
	ValidatorsBlockRewardsPoolName Pool = "validators_block_rewards_pool"
)

// refill reward pools time stores constant
// used to fill the block rewards pools (for both validators and providers)
// once a month.
const (
	RefillRewardsPoolTimerPrefix = "refill-rewards-pool-ts"
)
