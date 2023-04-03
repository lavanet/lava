package types

const (
	// ModuleName defines the module name
	ModuleName = "project"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_project"

	// prefix for the projects fixation store
	ProjectsFixationPrefix = "prj-fs"

	// prefix for the developer keys fixation store
	DeveloperKeysFixationPrefix = "dev-fs"

	// prefix for the subscription-to-projects mapping
	SubscriptionProjectsPrefix = "sub/projs/"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// SubscriptionKey returns the store key to retrieve a Subscription from the consumer field
func SubscriptionKey(consumer string) []byte {
	var key []byte

	indexBytes := []byte(consumer)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
