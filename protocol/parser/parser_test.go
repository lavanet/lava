package parser

import (
	"reflect"
	"testing"

	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

// TestAppendInterfaceToInterfaceArray tests append interface function
func TestAppendInterfaceToInterfaceArray(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []interface{}
	}{
		{
			name:     "Test with int value",
			input:    1,
			expected: []interface{}{1},
		},
		{
			name:     "Test with string value",
			input:    "hello",
			expected: []interface{}{"hello"},
		},
		{
			name:     "Test with struct value",
			input:    struct{ name string }{name: "John Doe"},
			expected: []interface{}{struct{ name string }{name: "John Doe"}},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := appendInterfaceToInterfaceArray(test.input)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Expected %v but got %v", test.expected, result)
			}
		})
	}
}

// TestParseArrayOfInterfaces tests parsing array of interfaces
func TestParseArrayOfInterfaces(t *testing.T) {
	tests := []struct {
		name     string
		data     []interface{}
		propName string
		sep      string
		expected []interface{}
	}{
		{
			name:     "Test with matching prop name",
			data:     []interface{}{"name:John Doe", "age:30", "gender:male"},
			propName: "name",
			sep:      ":",
			expected: []interface{}{"John Doe"},
		},
		{
			name:     "Test with non-matching prop name",
			data:     []interface{}{"name:John Doe", "age:30", "gender:male"},
			propName: "address",
			sep:      ":",
			expected: nil,
		},
		{
			name:     "Test with empty data array",
			data:     []interface{}{},
			propName: "name",
			sep:      ":",
			expected: nil,
		},
		{
			name:     "Test with non-string value in data array",
			data:     []interface{}{"name:John Doe", 30, "gender:male"},
			propName: "name",
			sep:      ":",
			expected: []interface{}{"John Doe"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := parseArrayOfInterfaces(test.data, test.propName, test.sep)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Expected %v but got %v", test.expected, result)
			}
		})
	}
}

func TestParseResponseByEncoding(t *testing.T) {
	type data struct {
		bytes    []byte
		encoding string
	}

	testInputs := func(testData []data) {
		result0, err := parseResponseByEncoding(testData[0].bytes, testData[0].encoding)
		require.NoError(t, err)
		result1, err := parseResponseByEncoding(testData[1].bytes, testData[1].encoding)
		require.NoError(t, err)
		require.Equal(t, result0, result1)
	}
	// returned from lava blockchain rest vs tendermintrpc
	testData := []data{{bytes: []byte("9291EDC036AE254F9A6E0237F0EF13C452E7F08722E8DBD68B2F34CC8132C91D"), encoding: spectypes.EncodingHex}, {bytes: []byte("kpHtwDauJU+abgI38O8TxFLn8Ici6NvWiy80zIEyyR0="), encoding: spectypes.EncodingBase64}}
	testInputs(testData)
	// returned form evmos evm-jsonrpc vs rest
	testData = []data{{bytes: []byte("0x968ec00fd34eedc03b0577ee8116f74c75127b7d775e51c7a72519f760b821a8"), encoding: spectypes.EncodingHex}, {bytes: []byte("lo7AD9NO7cA7BXfugRb3THUSe313XlHHpyUZ92C4Iag="), encoding: spectypes.EncodingBase64}}
	testInputs(testData)
}
