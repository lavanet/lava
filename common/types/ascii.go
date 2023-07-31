package types

import "unicode"

const (
	ASCII_MIN = 32  // min visible ascii
	ASCII_MAX = 126 // max visible ascii
	ASCII_DEL = 127 // ascii for DEL
)

type charRestrictionEnum string

const (
	NAME_RESTRICTIONS charRestrictionEnum = "name"
)

func isCharDisallowed(c rune, disallowedChars []rune) bool {
	for _, char := range disallowedChars {
		if char == c {
			return true
		}
	}
	return false
}

// Validates name strings.
// Current policy:
//
//	name: lowercase ascii letters and digits only and the characters {' ', '_'}. can't be empty.
func ValidateString(s string, restrictType charRestrictionEnum, disallowedChars []rune) bool {
	if restrictType == NAME_RESTRICTIONS && len(s) == 0 {
		return false
	}

	for _, r := range s {
		if disallowedChars != nil && isCharDisallowed(r, disallowedChars) {
			return false
		} else {
			switch restrictType {
			case NAME_RESTRICTIONS:
				if r == ',' {
					return false
				} else if !unicode.IsLower(r) && r != ' ' && r != '_' && !unicode.IsDigit(r) {
					return false
				}
			}
		}
	}

	return true
}
