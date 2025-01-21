package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/testutil/common"
	"github.com/lavanet/lava/v4/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// TestProviderReputation tests the provider-reputation query
func TestProviderReputation(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(4, 0, 0) // 4 providers

	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	_, p2 := ts.GetAccount(common.PROVIDER, 1)
	_, p3 := ts.GetAccount(common.PROVIDER, 2)
	_, p4 := ts.GetAccount(common.PROVIDER, 3)

	specs := []string{"spec1", "spec2", "spec1", "spec1", "spec1", "spec2", "spec2"}
	clusters := []string{"cluster1", "cluster1", "cluster2", "cluster1", "cluster1", "cluster1", "cluster2"}
	providers := []string{p1, p1, p1, p2, p3, p3, p4}

	// Reputation score setup:
	// spec1 + cluster1: p1=1, p2=4, p3=5
	// spec1 + cluster2: p1=3
	// spec2 + cluster1: p1=2, p3=6
	// spec2 + cluster2: p4=7
	for i := range providers {
		err := ts.Keepers.Pairing.SetReputationScore(ts.Ctx, specs[i], clusters[i], providers[i], sdk.NewDec(int64(i+1)))
		require.NoError(t, err)
	}

	// test only on p1
	tests := []struct {
		name     string
		chain    string
		cluster  string
		expected []types.ReputationData
	}{
		{
			"spec1+cluster1", "spec1", "cluster1", []types.ReputationData{
				{Rank: 3, Providers: 3, OverallPerformance: "bad", ChainID: "spec1", Cluster: "cluster1"},
			},
		},
		{
			"spec1", "spec1", "*", []types.ReputationData{
				{Rank: 3, Providers: 3, OverallPerformance: "bad", ChainID: "spec1", Cluster: "cluster1"},
				{Rank: 1, Providers: 1, OverallPerformance: "low_variance", ChainID: "spec1", Cluster: "cluster2"},
			},
		},
		{
			"cluster1", "*", "cluster1", []types.ReputationData{
				{Rank: 3, Providers: 3, OverallPerformance: "bad", ChainID: "spec1", Cluster: "cluster1"},
				{Rank: 2, Providers: 2, OverallPerformance: "bad", ChainID: "spec2", Cluster: "cluster2"},
			},
		},
		{
			"all", "*", "*", []types.ReputationData{
				{Rank: 3, Providers: 3, OverallPerformance: "bad", ChainID: "spec1", Cluster: "cluster1"},
				{Rank: 2, Providers: 2, OverallPerformance: "bad", ChainID: "spec2", Cluster: "cluster2"},
				{Rank: 1, Providers: 1, OverallPerformance: "low_variance", ChainID: "spec1", Cluster: "cluster2"},
			},
		},
		{
			"spec2+cluster2 (p1 not exist)", "spec2", "cluster2", []types.ReputationData{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ts.QueryPairingProviderReputation(p1, tt.chain, tt.cluster)
			require.NoError(t, err)
			for _, data := range res.Data {
				foundChainCluster := false
				for _, expected := range tt.expected {
					if data.ChainID == expected.ChainID && data.Cluster == expected.Cluster {
						foundChainCluster = true
						require.Equal(t, expected.Rank, data.Rank)
						require.Equal(t, expected.Providers, data.Providers)
						require.Equal(t, expected.OverallPerformance, data.OverallPerformance)
					}
				}
				if !foundChainCluster {
					require.FailNow(t, "could not find chain cluster pair on-chain")
				}
			}
		})
	}
}

// TestProviderReputationDetails tests the provider-reputation-details query
func TestProviderReputationDetails(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 0, 0) // 2 providers

	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	_, p2 := ts.GetAccount(common.PROVIDER, 1)

	specs := []string{"spec1", "spec2", "spec1", "spec1"}
	clusters := []string{"cluster1", "cluster1", "cluster2", "cluster1"}
	providers := []string{p1, p1, p1, p2}

	for i := range providers {
		ts.Keepers.Pairing.SetReputation(ts.Ctx, specs[i], clusters[i], providers[i], types.Reputation{
			Stake: sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(int64(i+1))),
		})
		err := ts.Keepers.Pairing.SetReputationScore(ts.Ctx, specs[i], clusters[i], providers[i], sdk.NewDec(int64(i+1)))
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		provider string
		chain    string
		cluster  string
		expected []math.LegacyDec
	}{
		{"provider+chain+cluster", p1, "spec1", "cluster1", []math.LegacyDec{math.LegacyNewDec(1)}},
		{"provider+chain+all_clusters", p1, "spec1", "*", []math.LegacyDec{math.LegacyNewDec(1), math.LegacyNewDec(3)}},
		{"provider+all_chain+cluster", p1, "*", "cluster1", []math.LegacyDec{math.LegacyNewDec(1), math.LegacyNewDec(2)}},
		{"provider+all_chains+all_clusters", p1, "*", "*", []math.LegacyDec{math.LegacyNewDec(1), math.LegacyNewDec(2), math.LegacyNewDec(3)}},
		{"second provider+chain+cluster", p2, "spec1", "cluster1", []math.LegacyDec{math.LegacyNewDec(4)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ts.QueryPairingProviderReputationDetails(tt.provider, tt.chain, tt.cluster)
			require.NoError(t, err)
			for i := range res.Data {
				expectedStake := sdk.NewCoin(ts.TokenDenom(), tt.expected[i].TruncateInt())
				expectedScore := tt.expected[i]

				require.Equal(t, tt.chain, res.Data[i].ChainID)
				require.Equal(t, tt.cluster, res.Data[i].Cluster)
				require.True(t, expectedStake.IsEqual(res.Data[i].Reputation.Stake))
				require.True(t, expectedScore.Equal(res.Data[i].ReputationPairingScore.Score))
			}
		})
	}
}
