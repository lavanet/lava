package rewardserver_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/testutil/common"
	"github.com/stretchr/testify/require"
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

func TestDelete(t *testing.T) {
	ps := rewardserver.NewProofStore()
	proof := common.BuildRelayRequest(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(0), "spec", nil)

	err := ps.Save(context.TODO(), "consumerKey", proof)
	require.NoError(t, err)

	err = ps.Delete(context.TODO(), proof.Epoch, "consumerKey")
	require.NoError(t, err)

	proofEntities, err := ps.FindAll(context.TODO())
	require.NoError(t, err)
	require.Equal(t, 0, len(proofEntities))
}

func TestDeleteAllForEpoch(t *testing.T) {
	ps := rewardserver.NewProofStore()
	proof1 := common.BuildRelayRequest(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(0), "spec", nil)
	proof2 := common.BuildRelayRequest(sdk.WrapSDKContext(newSdkContext()), "provider", []byte{}, uint64(0), "spec", nil)

	err := ps.Save(context.TODO(), "consumerKey", proof1)
	require.NoError(t, err)

	err = ps.Save(context.TODO(), "consumerKey", proof2)
	require.NoError(t, err)

	err = ps.DeleteAllForEpoch(context.TODO(), proof1.Epoch)
	require.NoError(t, err)

	proofEntities, err := ps.FindAll(context.TODO())
	require.NoError(t, err)
	require.Equal(t, 0, len(proofEntities))
}
