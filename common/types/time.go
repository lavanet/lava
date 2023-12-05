package types

import (
	"time"
)

const (
	TimeFormat = "2006-January-02 15:04:05"
)

func ConvertUnixTimestampToString(val uint64) string {
	return time.Unix(int64(val), 0).Format(TimeFormat)
}
