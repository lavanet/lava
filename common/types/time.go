package types

import (
	"time"

	"github.com/lavanet/lava/utils"
)

const (
	TimeFormat = "2006-January-02 15:04:05"
)

func ConvertUnixTimestampToString(val uint64) string {
	return time.Unix(int64(val), 0).Format(TimeFormat)
}

// NextMonth returns the date of the same day next month (assumes UTC),
// adjusting for end-of-months differences if needed.
func NextMonth(date time.Time) time.Time {
	// End-of-month days are tricky because months differ in days counts.
	// To avoid this complixity, we trim day-of-month greater than 28 back to
	// day 28, which all months always have (at the cost of the user possibly
	// losing 1 (and up to 3) days of subscription in the first month.

	if utils.DebugPaymentE2E == "debug_payment_e2e" {
		return time.Date(
			date.Year(),
			date.Month(),
			date.Day(),
			date.Hour(),
			date.Minute()+2,
			date.Second(),
			0,
			time.UTC,
		)
	}

	dayOfMonth := date.Day()
	if dayOfMonth > 28 {
		dayOfMonth = 28
	}

	return time.Date(
		date.Year(),
		date.Month()+1,
		dayOfMonth,
		date.Hour(),
		date.Minute(),
		date.Second(),
		0,
		time.UTC,
	)
}
