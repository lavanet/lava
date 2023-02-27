package types

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	EntryIndexKey string = "Entry_Index_"
	EntryKey      string = "Entry_Value_"
)
