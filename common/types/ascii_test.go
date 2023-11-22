package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringValidation(t *testing.T) {
	tests := []struct {
		name            string
		s               string
		restrictType    charRestrictionEnum
		disallowedChars []rune
		valid           bool
	}{
		// name restrictions tests
		{"valid_name", "hello", NAME_RESTRICTIONS, nil, true},
		{"valid_name_with_space", "hel lo", NAME_RESTRICTIONS, nil, true},
		{"valid_name_with_underscore", "hel_lo", NAME_RESTRICTIONS, nil, true},
		{"valid_name_with_digit", "hel2lo", NAME_RESTRICTIONS, nil, true},
		{"invalid_name_not_lowercase", "hEllo", NAME_RESTRICTIONS, nil, false},
		{"invalid_name_not_ascii", "he¢llo", NAME_RESTRICTIONS, nil, false},
		{"invalid_name_with_disallowed_char", "heallo", NAME_RESTRICTIONS, []rune{'a'}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ValidateString(tt.s, tt.restrictType, tt.disallowedChars)
			if tt.valid {
				require.True(t, res)
			} else {
				require.False(t, res)
			}
		})
	}
}

func TestByteSliceToASCIIStr(t *testing.T) {
	allChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ[]{}()!@#$%^&*'\"`~,."
	tests := []struct {
		input          string
		placeholder    rune
		expectedOutput string
	}{
		{
			"\x00\x12" + allChars + "\x88\x00",
			'@',
			"@@" + allChars + "@@",
		},
		{
			allChars,
			'@',
			allChars,
		},
	}

	for _, test := range tests {
		require.Equal(t, test.expectedOutput, ByteSliceToASCIIStr([]byte(test.input), test.placeholder))
	}
}
