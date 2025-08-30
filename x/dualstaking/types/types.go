package types

const (
	DelegateEventName          = "delegate_to_provider"
	UnbondingEventName         = "unbond_from_provider"
	RedelegateEventName        = "redelegate_between_providers"
	ClaimRewardsEventName      = "delegator_claim_rewards"
	ContributorRewardEventName = "contributor_rewards"
	ValidatorSlashEventName    = "validator_slash"
	FreezeFromUnbond           = "freeze_from_unbond"
	UnstakeFromUnbond          = "unstake_from_unbond"
	ProviderRewardEventName    = "provider_reward"
	DelegatorRewardEventName   = "delegator_reward"
)

const (
	NotBondedPoolName = "not_bonded_dualstaking_pool"
	BondedPoolName    = "bonded_dualstaking_pool"
)
