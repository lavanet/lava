package types

const (
	// ModuleName defines the module name
	ModuleName = "rewards"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_rewards"

	// prefix for the CU tracker timer store
	MonthlyRewardsTSPrefix = "monthly-rewards-ts"

	// prefix for IPRPC eligible subscriptions
	IprpcSubscriptionPrefix = "iprpc-subscription"

	// prefix for min IPRPC cost
	MinIprpcCostPrefix = "min-iprpc-cost"

	// prefix for IPRPC reward element
	IprpcRewardPrefix = "iprpc-reward"

	// prefix for IPRPC rewards count
	IprpcRewardsCountPrefix = "iprpc-rewards-count"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
