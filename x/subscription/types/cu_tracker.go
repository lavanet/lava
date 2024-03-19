package types

func (td CuTrackerTimerData) Validate() bool {
	if td.Block == 0 || td.Credit.Denom == "" {
		return false
	}
	return true
}
