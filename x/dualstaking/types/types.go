package types

const (
	DelegateEventName          = "delegate_to_provider"
	UnbondingEventName         = "unbond_from_provider"
	RedelegateEventName        = "redelegate_between_providers"
	RefundedEventName          = "refund_to_delegator"
	ClaimRewardsEventName      = "delegator_claim_rewards"
	ContributorRewardEventName = "contributor_rewards"
	ValidatorSlashEventName    = "validator_slash"
)

const (
	NotBondedPoolName = "not_bonded_dualstaking_pool"
	BondedPoolName    = "bonded_dualstaking_pool"
)
