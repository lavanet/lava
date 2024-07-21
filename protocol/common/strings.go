package common

import "strings"

func IsQuoted(s string) bool {
	return strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")
}

func IsSquareBracketed(s string) bool {
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}

func UnSquareBracket(s string) string {
	if IsSquareBracketed(s) {
		return s[1 : len(s)-1]
	}
	return s
}
