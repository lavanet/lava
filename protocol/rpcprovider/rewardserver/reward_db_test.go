package rewardserver_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

func TestSave(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB(db)
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)

	_, saved, err := rs.Save("consumerAddr", "consumerKey", proof)
	require.True(t, saved)
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)

	require.Equal(t, 1, len(rewards))
}

func TestFindAll(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB(db)
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)

	_, _, err := rs.Save("consumerAddr", "consumerKey"+"specId", proof)
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 1, len(rewards))
}

func TestFindOne(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB(db)
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)
	proof.Epoch = 1

	_, _, err := rs.Save("consumerAddr", "consumerKey", proof)
	require.NoError(t, err)

	reward, err := rs.FindOne(uint64(proof.Epoch), "consumerAddr", "consumerKey", proof.SessionId)
	require.NoError(t, err)
	require.NotNil(t, reward)
}

func TestDeleteClaimedRewards(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB(db)
	privKey, addr := sigs.GenerateFloatingKey()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))

	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)
	proof.Epoch = 1

	sig, err := sigs.Sign(privKey, *proof)
	require.NoError(t, err)
	proof.Sig = sig

	_, _, err = rs.Save(addr.String(), "consumerKey", proof)
	require.NoError(t, err)

	err = rs.DeleteClaimedRewards([]*pairingtypes.RelaySession{proof})
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 0, len(rewards))
}

func TestDeleteEpochRewards(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB(db)
	privKey, addr := sigs.GenerateFloatingKey()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))

	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)
	proof.Epoch = 1

	sig, err := sigs.Sign(privKey, *proof)
	require.NoError(t, err)
	proof.Sig = sig

	_, _, err = rs.Save(addr.String(), "consumerKey", proof)
	require.NoError(t, err)

	err = rs.DeleteEpochRewards(uint64(proof.Epoch))
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 0, len(rewards))
}

func TestRewardsWithTTL(t *testing.T) {
	db := rewardserver.NewMemoryDB("spec")
	// really really short TTL to make sure the rewards are not queryable
	ttl := 10 * time.Microsecond
	rs := rewardserver.NewRewardDBWithTTL(ttl)
	rs.AddDB(db)
	privKey, addr := sigs.GenerateFloatingKey()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))

	proof := common.BuildRelayRequest(ctx, "provider", []byte{}, uint64(0), "spec", nil)
	proof.Epoch = 1

	sig, err := sigs.Sign(privKey, *proof)
	require.NoError(t, err)
	proof.Sig = sig

	_, _, err = rs.Save(addr.String(), "consumerKey", proof)
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 0, len(rewards))
}
