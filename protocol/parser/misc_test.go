package parser

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLongestCommonSubsequenceWithFormat(t *testing.T) {
	playbook := []struct {
		s1       string
		s2       string
		expected string
		len      int
	}{
		{
			s1:       "123",
			s2:       "123",
			expected: "123",
			len:      3,
		},
		{
			s1:       "123 4",
			s2:       "123 ",
			expected: "123 %*s",
			len:      4,
		},
		{
			s1:       "123 ",
			s2:       "123 4",
			expected: "123 %*s",
			len:      4,
		},
		{
			s1:       "123 5",
			s2:       "123 4",
			expected: "123 %*s",
			len:      4,
		},
		{ // make sure the entire word is marked as a wildcard
			s1:       "1235",
			s2:       "1234",
			expected: "%*s",
			len:      0,
		},
		{ // make sure partial matches don't break it
			s1:       "123 456 789",
			s2:       "123 457 789",
			expected: "123 %*s 789",
			len:      8,
		},
		{
			s1:       "height 1234567 is too high current height is 01230123",
			s2:       "height 1234566 is too high current height is 01230124",
			expected: "height %*s is too high current height is %*s",
			len:      38,
		},
		{
			s1:       "1234567 is too high current height is 01230123 bro",
			s2:       "1234566 is too high current height is 01230124 bro",
			expected: "%*s is too high current height is %*s bro",
			len:      35,
		},
		{
			s1:       "1234567 is too high current height is 01230123 bro",
			s2:       "1234566 is too high current height is 01230124 nisan",
			expected: "%*s is too high current height is %*s %*s",
			len:      32,
		},
		{
			s1:       "yes5no",
			s2:       "yes6no",
			expected: "%*s",
			len:      0,
		},
		{
			s1:       "yes5no ",
			s2:       "yes6no ",
			expected: "%*s ",
			len:      1,
		},
		{
			s1:       "yes 5.no ",
			s2:       "yes 6.no ",
			expected: "yes %*s.no ",
			len:      8,
		},
	}
	for idx, play := range playbook {
		t.Run("Test "+strconv.Itoa(idx), func(t *testing.T) {
			expected, len := LongestCommonSubsequenceWithFormat(play.s1, play.s2)
			require.Equal(t, play.expected, expected)
			require.Equal(t, play.len, len)
		})
	}
}
