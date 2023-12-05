package types

// validators rewards pool constants
// this pool is used as the main reserve of token for validators rewards
// it gets its token through continuous vesting from the treatury account
const (
	ValidatorsPoolName = "validators_rewards_pool"
)

// validators block rewards pool constants
// this pool is used as the reserve for validator rewards per block
// it gets its token through the validators rewards pool each month
// this monthly transfer happens using the "refill block pool" timer store
const (
	ValidatorsBlockPoolName    = "validators_block_rewards_pool"
	RefillBlockPoolTimerPrefix = "refill-block-pool-ts"
)
