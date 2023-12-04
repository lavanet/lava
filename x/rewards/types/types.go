package types

import "time"

const (
	ValidatorsPoolName                           = "validators_rewards_pool"
	ValidatorsVestingTime          time.Duration = time.Hour * 24 * 30                // month
	ValidatorsTotalVestingTime     time.Duration = ValidatorsVestingTime * (12*4 - 1) // 4 years minus one month
	TestValidatorsVestingTime      time.Duration = time.Second
	TestValidatorsTotalVestingTime time.Duration = TestValidatorsVestingTime * (12*4 - 1)
)
