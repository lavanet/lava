package types

import (
	"bytes"
	"unicode"
)

const (
	ASCII_MIN           = 32  // min visible ascii
	ASCII_MAX           = 126 // max visible ascii
	ASCII_DEL           = 127 // ascii for DEL
	MAX_LEN_DESCRIPTION = 500
)

type charRestrictionEnum string

const (
	NAME_RESTRICTIONS        charRestrictionEnum = "name"
	DESCRIPTION_RESTRICTIONS charRestrictionEnum = "description"
	INDEX_RESTRICTIONS       charRestrictionEnum = "index"
)

func isCharDisallowed(c rune, disallowedChars []rune) bool {
	for _, char := range disallowedChars {
		if char == c {
			return true
		}
	}
	return false
}

func isASCII(r rune) bool {
	if r > 127 || !unicode.IsLetter(r) {
		return false
	}

	return true
}

func isLowercaseASCII(r rune) bool {
	return r >= 'a' && r <= 'z'
}

// Validates name strings.
// Current policy:
//
//	name: lowercase ascii letters and digits only and the characters {' ', '_'}. can't be empty.
//	description/reason: ascii letters (a-zA-z) and digits only and the characters {' ', '_'}. can't be empty. no more than MAX_PROJECT_DESCRIPTION_LEN.
func ValidateString(s string, restrictType charRestrictionEnum, disallowedChars []rune) bool {
	// Length check
	if restrictType == NAME_RESTRICTIONS && len(s) == 0 {
		return false
	}

	if restrictType == DESCRIPTION_RESTRICTIONS && (len(s) == 0 || len(s) > MAX_LEN_DESCRIPTION) {
		return false
	}

	if restrictType == INDEX_RESTRICTIONS && len(s) == 0 {
		return false
	}

	// Character check
	for _, r := range s {
		if disallowedChars != nil && isCharDisallowed(r, disallowedChars) {
			return false
		} else {
			switch restrictType {
			case NAME_RESTRICTIONS:
				if r == ',' {
					return false
				} else if !isLowercaseASCII(r) && r != ' ' && r != '_' && !unicode.IsDigit(r) {
					return false
				}
			case DESCRIPTION_RESTRICTIONS:
				if !isASCII(r) && r != ' ' && r != '_' && !unicode.IsDigit(r) {
					return false
				}
			case INDEX_RESTRICTIONS:
				if !isASCII(r) && !unicode.IsDigit(r) {
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
