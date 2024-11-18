package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetContentHashData_NilInputs(t *testing.T) {
	// Test case with all nil/empty values
	rp := RelayPrivateData{
		Metadata:       nil,
		Extensions:     nil,
		Addon:          "",
		ApiInterface:   "",
		ConnectionType: "",
		ApiUrl:         "",
		Data:           nil,
		Salt:           nil,
		RequestBlock:   0,
		SeenBlock:      0,
	}

	// Function should not panic and return a valid byte slice
	result := rp.GetContentHashData()
	require.NotNil(t, result)

	// Test with nil Extensions but non-nil empty slice
	rp.Extensions = []string{}
	result = rp.GetContentHashData()
	require.NotNil(t, result)

	// Test with nil Metadata but non-nil empty slice
	rp.Metadata = []Metadata{}
	result = rp.GetContentHashData()
	require.NotNil(t, result)
}

func TestGetContentHashData_NilData(t *testing.T) {
	// Test case with nil Data field specifically
	rp := RelayPrivateData{
		Metadata:       []Metadata{},
		Extensions:     []string{},
		Addon:          "",
		ApiInterface:   "",
		ConnectionType: "",
		ApiUrl:         "",
		Data:           nil, // Explicitly nil
		Salt:           make([]byte, 8),
		RequestBlock:   0,
		SeenBlock:      0,
	}

	// Function should not panic
	result := rp.GetContentHashData()
	require.NotNil(t, result)
}

func TestGetContentHashData_EdgeCases(t *testing.T) {
	testCases := []struct {
		name string
		rp   RelayPrivateData
	}{
		{
			name: "nil metadata with data",
			rp: RelayPrivateData{
				Metadata: nil,
				Data:     []byte("test"),
			},
		},
		{
			name: "nil extensions with data",
			rp: RelayPrivateData{
				Extensions: nil,
				Data:       []byte("test"),
			},
		},
		{
			name: "nil everything",
			rp: RelayPrivateData{
				Metadata:       nil,
				Extensions:     nil,
				Addon:          "",
				ApiInterface:   "",
				ConnectionType: "",
				ApiUrl:         "",
				Data:           nil,
				Salt:           nil,
			},
		},
		{
			name: "empty strings with nil slices",
			rp: RelayPrivateData{
				Metadata:       []Metadata{},
				Extensions:     []string{},
				Addon:          "",
				ApiInterface:   "",
				ConnectionType: "",
				ApiUrl:         "",
				Data:           nil,
				Salt:           nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Function should not panic
			result := tc.rp.GetContentHashData()
			require.NotNil(t, result)
		})
	}
}
