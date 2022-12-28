package types

const (
	ProviderStakeEventName       = "stake_new_provider"
	ConsumerStakeEventName       = "stake_new_consumer"
	ProviderStakeUpdateEventName = "stake_update_provider"
	ConsumerStakeUpdateEventName = "stake_update_consumer"
	ProviderUnstakeEventName     = "provider_unstake_commit"
	ConsumerUnstakeEventName     = "consumer_unstake_commit"

	RelayPaymentEventName                      = "relay_payment"
	UnresponsiveProviderUnstakeFailedEventName = "unresponsive_provider"
	ProviderJailedEventName                    = "provider_jailed"
)

func StakeNewEventName(isProvider bool) string {
	if isProvider {
		return ProviderStakeEventName
	} else {
		return ConsumerStakeEventName
	}
}

func StakeUpdateEventName(isProvider bool) string {
	if isProvider {
		return ProviderStakeUpdateEventName
	} else {
		return ConsumerStakeUpdateEventName
	}
}

func UnstakeCommitNewEventName(isProvider bool) string {
	if isProvider {
		return ProviderUnstakeEventName
	} else {
		return ConsumerUnstakeEventName
	}
}

type ClientUsedCU struct {
	TotalUsed uint64
	Providers map[string]uint64
}

type ClientProviderOverusedCUPercent struct {
	TotalOverusedPercent    float64
	OverusedPercentProvider float64
}
