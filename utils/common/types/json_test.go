package types_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lavanet/lava/v5/utils/common/types"
)

func TestCreateCanonicalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "Empty JSON object",
			input:   []byte(`{}`),
			want:    "{}",
			wantErr: false,
		},
		{
			name:    "Simple key-value pair",
			input:   []byte(`{"name": "test"}`),
			want:    `{"name":"test"}`,
			wantErr: false,
		},
		{
			name:    "Multiple key-value pairs",
			input:   []byte(`{"b": 2, "a": 1, "c": 3}`),
			want:    `{"a":1,"b":2,"c":3}`,
			wantErr: false,
		},
		{
			name:    "Nested objects",
			input:   []byte(`{"outer": {"inner2": 2, "inner1": 1}}`),
			want:    `{"outer":{"inner1":1,"inner2":2}}`,
			wantErr: false,
		},
		{
			name:    "Array values",
			input:   []byte(`{"array": [3, 1, 2], "value": "test"}`),
			want:    `{"array":[3,1,2],"value":"test"}`,
			wantErr: false,
		},
		{
			name: "Complex nested structure",
			input: []byte(`{
				"z": [{"b": 2, "a": 1}, {"d": 4, "c": 3}],
				"y": {"inner2": true, "inner1": false},
				"x": "value"
			}`),
			want:    `{"x":"value","y":{"inner1":false,"inner2":true},"z":[{"a":1,"b":2},{"c":3,"d":4}]}`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   []byte(`{invalid json}`),
			want:    "",
			wantErr: true,
		},
		{
			name: "Different types of values",
			input: []byte(`{
				"string": "value",
				"number": 42,
				"float": 3.14,
				"bool": true,
				"null": null,
				"array": [1, "two", 3.0, null, false]
			}`),
			want:    `{"array":[1,"two",3.0,null,false],"bool":true,"float":3.14,"null":null,"number":42,"string":"value"}`,
			wantErr: false,
		},
		{
			name:    "Empty array",
			input:   []byte(`{"arr":[]}`),
			want:    `{"arr":[]}`,
			wantErr: false,
		},
		{
			name:    "Unicode characters",
			input:   []byte(`{"emoji": "😀", "unicode": "こんにちは"}`),
			want:    `{"emoji":"😀","unicode":"こんにちは"}`,
			wantErr: false,
		},
		{
			name:    "Deeply nested objects",
			input:   []byte(`{"l1":{"l2":{"l3":{"l4":{"b":2,"a":1}}}}}`),
			want:    `{"l1":{"l2":{"l3":{"l4":{"a":1,"b":2}}}}}`,
			wantErr: false,
		},
		{
			name:    "Special characters in keys",
			input:   []byte(`{"special-key":1,"special_key":2,"special.key":3}`),
			want:    `{"special-key":1,"special.key":3,"special_key":2}`,
			wantErr: false,
		},
		{
			name:    "Empty string values",
			input:   []byte(`{"empty":"","notempty":"value"}`),
			want:    `{"empty":"","notempty":"value"}`,
			wantErr: false,
		},
		{
			name:    "Numbers with different formats",
			input:   []byte(`{"int":42,"scientific":1e2,"negative":-1,"decimal":3.14}`),
			want:    `{"decimal":3.14,"int":42,"negative":-1,"scientific":100}`,
			wantErr: false,
		},
		{
			name:    "Array with nested objects",
			input:   []byte(`{"arr":[{"c":3,"b":2,"a":1},{"f":6,"e":5,"d":4}]}`),
			want:    `{"arr":[{"a":1,"b":2,"c":3},{"d":4,"e":5,"f":6}]}`,
			wantErr: false,
		},
		{
			name:    "Mixed array types",
			input:   []byte(`{"arr":[1,"string",{"b":2,"a":1},null,[1,2,3],true]}`),
			want:    `{"arr":[1,"string",{"a":1,"b":2},null,[1,2,3],true]}`,
			wantErr: false,
		},
		{
			name:    "Invalid UTF-8",
			input:   []byte{0x7b, 0x22, 0x6b, 0x22, 0x3a, 0xff, 0x7d},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := types.CreateCanonicalJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCanonicalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Normalize both strings for comparison
				normalizedGot, err := types.CreateCanonicalJSON([]byte(got))
				if err != nil {
					t.Errorf("Failed to normalize got result: %v", err)
					return
				}
				normalizedWant, err := types.CreateCanonicalJSON([]byte(tt.want))
				if err != nil {
					t.Errorf("Failed to normalize want result: %v", err)
					return
				}

				if normalizedGot != normalizedWant {
					t.Errorf("CreateCanonicalJSON()\ngot  = %v\nwant = %v", got, tt.want)
				}
			}
		})
	}
}

// TestCreateCanonicalJSONWithLargeInput tests the function with a larger input
func TestCreateCanonicalJSONWithLargeInput(t *testing.T) {
	// Create a large JSON object
	largeObj := make(map[string]interface{})
	for i := 'z'; i >= 'a'; i-- {
		largeObj[string(i)] = i
	}

	input, err := json.Marshal(largeObj)
	if err != nil {
		t.Fatalf("Failed to create test input: %v", err)
	}

	got, err := types.CreateCanonicalJSON(input)
	if err != nil {
		t.Fatalf("CreateCanonicalJSON() error = %v", err)
	}

	// Check if the string contains keys in order
	lastIndex := -1
	for c := 'a'; c <= 'z'; c++ {
		currentIndex := strings.Index(got, string(c))
		if currentIndex == -1 {
			t.Errorf("Missing key %c in result", c)
			continue
		}
		if currentIndex < lastIndex {
			t.Errorf("Key %c appears before previous key in JSON string", c)
		}
		lastIndex = currentIndex
	}
}

func TestCreateCanonicalForm(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "Valid JSON input",
			input:   []byte(`{"b": 2, "a": 1}`),
			want:    `{"a":1,"b":2}`,
			wantErr: false,
		},
		{
			name:    "Binary data (protobuf-like)",
			input:   []byte{0x0a, 0x48, 0x0a, 0x20, 0x72, 0x39},
			want:    "0a480a207239", // hex encoding of the binary data
			wantErr: false,
		},
		{
			name:    "Empty binary data",
			input:   []byte{},
			want:    "",
			wantErr: false,
		},
		{
			name:    "Mixed control characters (non-JSON)",
			input:   []byte{0x00, 0x01, 0x02, 0xff},
			want:    "000102ff", // hex encoding
			wantErr: false,
		},
		{
			name:    "Complex nested JSON",
			input:   []byte(`{"z": [{"b": 2, "a": 1}], "y": {"inner2": true, "inner1": false}}`),
			want:    `{"y":{"inner1":false,"inner2":true},"z":[{"a":1,"b":2}]}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := types.CreateCanonicalForm(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCanonicalForm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("CreateCanonicalForm()\ngot  = %v\nwant = %v", got, tt.want)
			}
		})
	}
}

// TestCreateCanonicalFormConsistency tests that the same data always produces the same canonical form
func TestCreateCanonicalFormConsistency(t *testing.T) {
	testCases := [][]byte{
		[]byte(`{"b": 2, "a": 1, "c": 3}`),
		{0x0a, 0x48, 0x0a, 0x20, 0x72, 0x39, 0xf6, 0x46, 0x2e, 0xdb, 0xaa},
		[]byte("simple text"),
	}

	for i, testData := range testCases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			// Call multiple times and ensure we get the same result
			result1, err1 := types.CreateCanonicalForm(testData)
			result2, err2 := types.CreateCanonicalForm(testData)
			result3, err3 := types.CreateCanonicalForm(testData)

			if err1 != nil || err2 != nil || err3 != nil {
				t.Errorf("CreateCanonicalForm() returned unexpected errors: %v, %v, %v", err1, err2, err3)
				return
			}

			if result1 != result2 || result2 != result3 {
				t.Errorf("CreateCanonicalForm() not consistent: %v, %v, %v", result1, result2, result3)
			}
		})
	}
}
