package types

const (
	// SubscriptionKeyPrefix is the prefix to retrieve all Subscription
	SubscriptionKeyPrefix = "Subscribe/value/"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
