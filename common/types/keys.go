package types

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	UniqueIndexKey      = "UniqueIndex_value_"
	UniqueIndexCountKey = "UniqueIndex_count_"
)
