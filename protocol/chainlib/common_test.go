package chainlib

import (
	"testing"

	spectypes "github.com/lavanet/lava/x/spec/types"
)

func TestMatchSpecApiByName(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name        string
		serverApis  map[string]spectypes.ServiceApi
		inputName   string
		expectedApi spectypes.ServiceApi
		expectedOk  bool
	}{
		{
			name: "test1",
			serverApis: map[string]spectypes.ServiceApi{
				"test1.*": {Name: "test1-api"},
				"test2.*": {Name: "test2-api"},
			},
			inputName:   "test1-match",
			expectedApi: spectypes.ServiceApi{Name: "test1-api"},
			expectedOk:  true,
		},
		{
			name: "test2",
			serverApis: map[string]spectypes.ServiceApi{
				"test1.*": {Name: "test1-api"},
				"test2.*": {Name: "test2-api"},
			},
			inputName:   "test2-match",
			expectedApi: spectypes.ServiceApi{Name: "test2-api"},
			expectedOk:  true,
		},
		{
			name: "test3",
			serverApis: map[string]spectypes.ServiceApi{
				"test1.*": {Name: "test1-api"},
				"test2.*": {Name: "test2-api"},
			},
			inputName:   "test3-match",
			expectedApi: spectypes.ServiceApi{},
			expectedOk:  false,
		},
	}
	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			api, ok := matchSpecApiByName(testCase.inputName, testCase.serverApis)
			if ok != testCase.expectedOk {
				t.Fatalf("expected ok value %v, but got %v", testCase.expectedOk, ok)
			}
			if api.Name != testCase.expectedApi.Name {
				t.Fatalf("expected api %v, but got %v", testCase.expectedApi.Name, api.Name)
			}
		})
	}
}

func TestConvertToJsonError(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name     string
		errorMsg string
		expected string
	}{
		{
			name:     "valid json",
			errorMsg: "some error message",
			expected: `{"error":"some error message"}`,
		},
	}

	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := convertToJsonError(testCase.errorMsg)
			if result != testCase.expected {
				t.Errorf("Expected result to be %s, but got %s", testCase.expected, result)
			}
		})
	}
}

func TestAddAttributeToError(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name         string
		key          string
		value        string
		errorMessage string
		expected     string
	}{
		{
			name:         "Valid conversion",
			key:          "key1",
			value:        "value1",
			errorMessage: `"errorKey": "error_value"`,
			expected:     `"errorKey": "error_value", "key1": "value1"`,
		},
	}

	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := addAttributeToError(testCase.key, testCase.value, testCase.errorMessage)
			if result != testCase.expected {
				t.Errorf("addAttributeToError(%q, %q, %q) = %q; expected %q", testCase.key, testCase.value, testCase.errorMessage, result, testCase.expected)
			}
		})
	}
}
