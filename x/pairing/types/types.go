package types

const (
	ProviderStakeEventName       = "stake_new_provider"
	ProviderStakeUpdateEventName = "stake_update_provider"
	ProviderUnstakeEventName     = "provider_unstake_commit"

	RelayPaymentEventName            = "relay_payment"
	ProviderTemporaryJailedEventName = "provider_temporary_jailed"
	ProviderFreezeJailedEventName    = "provider_jailed"
	ProviderReportedEventName        = "provider_reported"
	LatestBlocksReportEventName      = "provider_latest_block_report"
	RejectedCuEventName              = "rejected_cu"
	UnstakeProposalEventName         = "unstake_gov_proposal"
)

// unstake description strings
const (
	UnstakeDescriptionClientUnstake     = "Client unstaked entry"
	UnstakeDescriptionProviderUnstake   = "Provider unstaked entry"
	UnstakeDescriptionInsufficientFunds = "client stake is below the minimum stake required"
)

const (
	FlagMoniker                  = "provider-moniker"
	FlagCommission               = "delegate-commission"
	FlagDelegationLimit          = "delegate-limit"
	FlagProvider                 = "provider"
	FlagIdentity                 = "identity"
	FlagWebsite                  = "website"
	FlagSecurityContact          = "security-contact"
	FlagDescriptionDetails       = "description-details"
	FlagGrantFeeAuth             = "grant-provider-gas-fees-auth"
	MAX_ENDPOINTS_AMOUNT_PER_GEO = 5 // max number of endpoints per geolocation for provider stake entry
)

// unresponsiveness consts
const (
	// Consider changing back on mainnet when providers QoS benchmarks are better // EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER uint64 = 4 // number of epochs to sum CU that the provider serviced
	EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER uint64 = 8 // number of epochs to sum CU that the provider serviced
	EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS              uint64 = 2 // number of epochs to sum CU of complainers against the provider
)
