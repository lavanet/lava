package parser

import (
	"reflect"
	"testing"
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
