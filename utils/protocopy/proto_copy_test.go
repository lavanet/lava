package protocopy

import (
	"testing"

	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	request := &pairingtypes.RelayPrivateData{
		ConnectionType: "test1",
		Data:           []byte{1, 2, 3, 4, 5},
		ApiUrl:         "test2",
		Addon:          "test3",
		RequestBlock:   3,
		ApiInterface:   "test4",
		Salt:           []byte{1, 2, 3, 4, 5, 6},
		Extensions:     []string{"test1", "test2", "test3", "test4"},
		SeenBlock:      4,
	}
	response := &pairingtypes.RelayPrivateData{}
	err := DeepCopyProtoObject(request, response)
	require.NoError(t, err)
	// check the values match between request and response.
	require.Equal(t, request.ConnectionType, response.ConnectionType)
	require.Equal(t, request.Data, response.Data)
	require.Equal(t, request.ApiUrl, response.ApiUrl)
	require.Equal(t, request.Addon, response.Addon)
	require.Equal(t, request.RequestBlock, response.RequestBlock)
	require.Equal(t, request.ApiInterface, response.ApiInterface)
	require.Equal(t, request.Salt, response.Salt)
	require.Equal(t, request.Extensions, response.Extensions)
	require.Equal(t, request.SeenBlock, response.SeenBlock)
}
