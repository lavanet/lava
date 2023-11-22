package types

import (
	"bytes"
	"unicode"
)

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

// Convert byte slice to ASCII-only string
// Non-ASCII characters will be replaced with placeholder
func ByteSliceToASCIIStr(input []byte, placeholder rune) string {
	var result bytes.Buffer

	for _, b := range input {
		if b >= ASCII_MIN && b <= ASCII_MAX {
			// If the byte is within the ASCII printable range, add it to the result.
			result.WriteByte(b)
		} else {
			// If it's not an ASCII printable character, replace it with the placeholder.
			result.WriteRune(placeholder)
		}
	}

	return result.String()
}
