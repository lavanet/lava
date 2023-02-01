package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// getDummyRequest creates dummy request used in tests
func createDummyRequest(requestedBlock int64, dataReliability *VRFData) *RelayRequest {
	return &RelayRequest{
		ChainID:         "testID",
		Data:            []byte("Dummy data"),
		RequestBlock:    requestedBlock,
		DataReliability: dataReliability,
	}
}

// TestRelayShallowCopy tests shallow copy method of relay request
func TestRelayShallowCopy(t *testing.T) {
	t.Parallel()
	t.Run(
		"Check if copy object has same data as original",
		func(t *testing.T) {
			t.Parallel()

			dataReliability := &VRFData{
				Differentiator: true,
			}

			request := createDummyRequest(-2, dataReliability)
			copy := request.ShallowCopy()

			assert.Equal(t, request, copy)
		})
	t.Run(
		"Only nested values should be shared",
		func(t *testing.T) {
			t.Parallel()

			dataReliability := &VRFData{
				Differentiator: true,
			}

			requestedBlock := int64(-2)

			request := createDummyRequest(requestedBlock, dataReliability)
			copy := request.ShallowCopy()

			// Change RequestBlock
			copy.RequestBlock = 1000

			// Check that Requested block has not changed in the original
			assert.Equal(t, request.RequestBlock, requestedBlock)

			// Change shared dataReliability
			dataReliability.Differentiator = false

			// DataReliability should be changed on both objects
			assert.Equal(t, request.DataReliability, copy.DataReliability)
		})
}
