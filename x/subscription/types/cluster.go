package types

const FREE_PLAN = "free" // gets its own const because it's treated differently

func GetSubUsageCriterion() []uint64 {
	// 0 = under a month, 6 = between 1-6 months, 7 = over 6 months
	return []uint64{0, 6, 7}
}
