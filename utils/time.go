package utils

import "time"

const MONTHS_IN_YEAR = 12

// NextMonth returns the date of the same day next month (assumes UTC),
// adjusting for end-of-months differences if needed.
func NextMonth(date time.Time) time.Time {
	// End-of-month days are tricky because months differ in days counts.
	// To avoid this complexity, we trim day-of-month greater than 28 back to
	// day 28, which all months always have (at the cost of the user possibly
	// losing 1 (and up to 3) days of subscription in the first month.

	if DebugPaymentE2E == "debug_payment_e2e" {
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

func IsMiddleOfMonthPassed(date time.Time) bool {
	// Get the total number of days in the current month
	_, month, year := date.Date()
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	// Calculate the middle day of the month
	middleDay := daysInMonth / 2

	// Check if the day of the given date is greater than the middle day
	return date.Day() > middleDay
}
