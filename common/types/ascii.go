package types

import "unicode"

const (
	ASCII_MIN = 32  // min visible ascii
	ASCII_MAX = 126 // max visible ascii
	ASCII_DEL = 127 // ascii for DEL
)

func IsASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// sanitizeIdnex checks that a string contains only visible ascii characters
// (i.e. Ascii 32-126), and appends a (ascii) DEL to the index; this ensures
// that an index can never be a prefix of another index.
func SanitizeIndex(index string) (string, error) {
	for i := 0; i < len(index); i++ {
		if index[i] < ASCII_MIN || index[i] > ASCII_MAX {
			return index, ErrInvalidIndex
		}
	}
	return index + string([]byte{ASCII_DEL}), nil
}

// desantizeIndex reverts the effect of sanitizeIndex - removes the trailing
// (ascii) DEL terminator.
func DesanitizeIndex(safeIndex string) string {
	return safeIndex[0 : len(safeIndex)-1]
}

func AssertSanitizedIndex(safeIndex string, prefix string) {
	if []byte(safeIndex)[len(safeIndex)-1] != ASCII_DEL {
		panic("Fixation: prefix " + prefix + ": unsanitized safeIndex: " + safeIndex)
	}
}
