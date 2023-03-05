package types

import (
	"strings"
)

func stringPlan(planToString *Plan, b strings.Builder) strings.Builder {
	b.WriteString(planToString.String())
	return b
}
