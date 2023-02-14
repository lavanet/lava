package types

import (
	"strings"
)

func stringPackage(packageToString *Package, b strings.Builder) strings.Builder {
	b.WriteString(packageToString.String())
	return b
}
