package types

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	UniqueIndexKey      = "UniqueIndex-value-"
	UniqueIndexCountKey = "UniqueIndex-count-"
)
