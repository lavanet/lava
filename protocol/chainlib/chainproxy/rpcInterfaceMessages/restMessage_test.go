package rpcInterfaceMessages

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestMessage(t *testing.T) {
	// Test GetParams method
	restMessage := RestMessage{
		Path:     "blocks/latest",
		SpecPath: "blocks/latest",
	}

	// Test GetParams method
	params := restMessage.GetParams()
	require.Nil(t, params)

	// Test GetResult method
	result := restMessage.GetResult()
	if result != nil {
		t.Errorf("Expected nil, but got %v", result)
	}
}

func TestRestParseBlock(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "Default block param",
			input:    "latest",
			expected: -2,
		},
		{
			name:     "String representation of int64",
			input:    "80",
			expected: 80,
		},
		{
			name:     "Hex representation of int64",
			input:    "0x26D",
			expected: 621,
		},
	}

	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			restMessage := RestMessage{}

			block, err := restMessage.ParseBlock(testCase.input)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if block != testCase.expected {
				t.Errorf("Expected %v, but got %v", testCase.expected, block)
			}
		})
	}
}
