package rewardserver_test

import (
	"testing"

	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestSave(t *testing.T) {
	ts := setup(t)

	db := rewardserver.NewMemoryDB()
	rs := rewardserver.NewRewardDB(db)
	proof := common.BuildRelayRequest(ts.ctx, "provider", []byte{}, uint64(0), "spec", nil)

	saved, err := rs.Save("consumerAddr", "consumerKey", proof)
	require.True(t, saved)
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)

	require.Equal(t, 1, len(rewards))
}

func TestFindOne(t *testing.T) {
	ts := setup(t)

	db := rewardserver.NewMemoryDB()
	rs := rewardserver.NewRewardDB(db)
	proof := common.BuildRelayRequest(ts.ctx, "provider", []byte{}, uint64(0), "spec", nil)
	proof.Epoch = 1

	_, err := rs.Save("consumerAddr", "consumerKey", proof)
	require.NoError(t, err)

	reward, err := rs.FindOne(uint64(proof.Epoch), "consumerAddr", "consumerKey", proof.SessionId)
	require.NoError(t, err)
	require.NotNil(t, reward)
}

func TestDeleteClaimedRewards(t *testing.T) {
	ts := setup(t)

	db := rewardserver.NewMemoryDB()
	rs := rewardserver.NewRewardDB(db)

	proof := common.BuildRelayRequest(ts.ctx, "provider", []byte{}, uint64(0), "spec", nil)
	proof.Epoch = 1

	sig, err := sigs.SignRelay(ts.consumer.SK, *proof)
	require.NoError(t, err)
	proof.Sig = sig

	_, err = rs.Save(ts.consumer.Addr.String(), "consumerKey", proof)
	require.NoError(t, err)

	err = rs.DeleteClaimedRewards([]*pairingtypes.RelaySession{proof})
	require.NoError(t, err)

	rewards, err := rs.FindAll()
	require.NoError(t, err)
	require.Equal(t, 0, len(rewards))
}
