package rewardserver_test

import (
	"fmt"
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestSave(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB()
	err := rs.AddDB(db)
	require.NoError(t, err)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)

	_, saved, err := rs.Save("consumerAddr", "consumerKey", proof)
	require.True(t, saved)
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)

	require.Equal(t, 1, len(rewards))
}

func TestSaveThreshold(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB()
	err := rs.AddDB(db)
	require.NoError(t, err)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)

	for i := 0; i < 101; i++ {
		_, saved, err := rs.Save(fmt.Sprintf("consumerAddr%d", i), fmt.Sprintf("consumerKey%d", i), proof)
		require.NoError(t, err)
		require.True(t, saved)
	}

	rewards, err := rs.FindAll()
	require.NoError(t, err)

	require.Equal(t, 1, len(rewards))
	require.Equal(t, 101, rewards[0].NumRewards())
}

func TestFindAll(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB()
	err := rs.AddDB(db)
	require.NoError(t, err)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)

	_, saved, err := rs.Save("consumerAddr", "consumerKey"+"specId", proof)
	require.NoError(t, err)
	require.True(t, saved)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 1, len(rewards))
}

func TestFindOne(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB()
	err := rs.AddDB(db)
	require.NoError(t, err)

	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))
	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)
	proof.Epoch = 1

	_, saved, err := rs.Save("consumerAddr", "consumerKey", proof)
	require.NoError(t, err)
	require.True(t, saved)

	reward, err := rs.FindOne(uint64(proof.Epoch), "consumerAddr", "consumerKey", proof.SessionId)
	require.NoError(t, err)
	require.NotNil(t, reward)
}

func TestDeleteClaimedRewards(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB()
	err := rs.AddDB(db)
	require.NoError(t, err)

	privKey, addr := sigs.GenerateFloatingKey()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))

	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)
	proof.Epoch = 1

	sig, err := sigs.Sign(privKey, *proof)
	require.NoError(t, err)
	proof.Sig = sig

	_, saved, err := rs.Save(addr.String(), "consumerKey", proof)
	require.NoError(t, err)
	require.True(t, saved)

	err = rs.DeleteClaimedRewards([]*pairingtypes.RelaySession{proof})
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 0, len(rewards))
}

func TestDeleteEpochRewards(t *testing.T) {
	db := rewardserver.NewMemoryDB("specId")
	rs := rewardserver.NewRewardDB()
	err := rs.AddDB(db)
	require.NoError(t, err)

	privKey, addr := sigs.GenerateFloatingKey()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))

	proof := common.BuildRelayRequest(ctx, "providerAddr", []byte{}, uint64(0), "specId", nil)
	proof.Epoch = 1

	sig, err := sigs.Sign(privKey, *proof)
	require.NoError(t, err)
	proof.Sig = sig

	_, saved, err := rs.Save(addr.String(), "consumerKey", proof)
	require.NoError(t, err)
	require.True(t, saved)

	err = rs.DeleteEpochRewards(uint64(proof.Epoch))
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 0, len(rewards))
}

func TestRewardsWithTTL(t *testing.T) {
	db := rewardserver.NewMemoryDB("spec")
	// really really short TTL to make sure the rewards are not queryable
	ttl := 1 * time.Nanosecond
	rs := rewardserver.NewRewardDBWithTTL(ttl)
	err := rs.AddDB(db)
	require.NoError(t, err)

	privKey, addr := sigs.GenerateFloatingKey()
	ctx := sdk.WrapSDKContext(sdk.NewContext(nil, tmproto.Header{}, false, nil))

	proof := common.BuildRelayRequest(ctx, "provider", []byte{}, uint64(0), "spec", nil)
	proof.Epoch = 1

	sig, err := sigs.Sign(privKey, *proof)
	require.NoError(t, err)
	proof.Sig = sig

	_, saved, err := rs.Save(addr.String(), "consumerKey", proof)
	require.NoError(t, err)
	require.True(t, saved)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 0, len(rewards))
}
