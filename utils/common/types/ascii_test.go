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
		{"invalid_name_not_ascii", "heあllo", NAME_RESTRICTIONS, nil, false},
		{"invalid_name_with_disallowed_char", "heallo", NAME_RESTRICTIONS, []rune{'a'}, false},
		{"invalid_empty_name", "", NAME_RESTRICTIONS, nil, false},

		// description restrictions tests
		{"valid_desc", "hello", DESCRIPTION_RESTRICTIONS, nil, true},
		{"valid_desc_with_space", "hel lo", DESCRIPTION_RESTRICTIONS, nil, true},
		{"valid_desc_with_underscore", "hel_lo", DESCRIPTION_RESTRICTIONS, nil, true},
		{"valid_desc_with_digit", "hel2lo", DESCRIPTION_RESTRICTIONS, nil, true},
		{"valid_desc_not_lowercase", "hEllo", DESCRIPTION_RESTRICTIONS, nil, true},
		{"invalid_desc_not_ascii", "heあllo", DESCRIPTION_RESTRICTIONS, nil, false},
		{"invalid_desc_with_disallowed_char", "heallo", DESCRIPTION_RESTRICTIONS, []rune{'a'}, false},
		{"invalid_empty_desc", "", DESCRIPTION_RESTRICTIONS, nil, false},

		// index restrictions tests
		{"valid_index", "hello", INDEX_RESTRICTIONS, nil, true},
		{"invalid_index_with_space", "hel lo", INDEX_RESTRICTIONS, nil, false},
		{"invalid_index_with_underscore", "hel_lo", INDEX_RESTRICTIONS, nil, false},
		{"valid_index_with_digit", "hel2lo", INDEX_RESTRICTIONS, nil, true},
		{"valid_index_not_lowercase", "hEllo", INDEX_RESTRICTIONS, nil, true},
		{"invalid_index_not_ascii", "heあllo", INDEX_RESTRICTIONS, nil, false},
		{"invalid_index_with_disallowed_char", "heallo", INDEX_RESTRICTIONS, []rune{'a'}, false},
		{"invalid_empty_index", "", INDEX_RESTRICTIONS, nil, false},
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
