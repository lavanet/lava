package rewardserver_test

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/testutil/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSave(t *testing.T) {
	ps := rewardserver.NewProofStore()
	proof := common.BuildRelayRequest(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(0), "spec", nil)

	err := ps.Save(context.TODO(), "consumerKey", proof)
	require.NoError(t, err)

	proofEntities, err := ps.FindAll(context.TODO())

	require.NoError(t, err)
	require.Equal(t, 1, len(proofEntities))
}
