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
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
