package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/stretchr/testify/require"
)

func TestAdjustment(t *testing.T) {
	ts := common.NewTester(t)
	err := ts.SetupForTests("../../../", "LAV1", 1, 2, 1, 2)
	require.NoError(t, err)
	_, consumer1Addr := ts.GetAccount(common.CONSUMER, 0)
	projects := ts.Keepers.Projects.GetAllProjectsForSubscription(ts.Ctx, consumer1Addr)
	require.Len(t, projects, 2)
	_, consumer2Addr := ts.GetAccount(common.CONSUMER, 1)
	projects = append(projects, ts.Keepers.Projects.GetAllProjectsForSubscription(ts.Ctx, consumer2Addr)...)
	require.Len(t, projects, 4)
	_, provider1 := ts.GetAccount(common.PROVIDER, 0)
	_, provider2 := ts.GetAccount(common.PROVIDER, 1)
	type indexConsumer struct {
		index int
		proj  string
	}
	usage := map[indexConsumer]struct {
		provider1Usage uint64
		provider2Usage uint64
	}{
		{
			index: 0,
			proj:  projects[0],
		}: {
			provider1Usage: 90,
			provider2Usage: 10,
		},
		{
			index: 1,
			proj:  projects[0],
		}: {
			provider1Usage: 180,
			provider2Usage: 20,
		},
		{
			index: 0,
			proj:  projects[1],
		}: {
			provider1Usage: 90,
			provider2Usage: 10,
		},
		{
			index: 1,
			proj:  projects[1],
		}: {
			provider1Usage: 180,
			provider2Usage: 20,
		},
		{
			index: 0,
			proj:  projects[2],
		}: {
			provider1Usage: 50,
			provider2Usage: 50,
		},
		{
			index: 1,
			proj:  projects[2],
		}: {
			provider1Usage: 100,
			provider2Usage: 100,
		},
		{
			index: 0,
			proj:  projects[3],
		}: {
			provider1Usage: 50,
			provider2Usage: 50,
		},
		{
			index: 1,
			proj:  projects[3],
		}: {
			provider1Usage: 100,
			provider2Usage: 1,
		},
	}

	for idxConsumer, cus := range usage {
		totalUsageThisIndex := cus.provider1Usage + cus.provider2Usage
		ts.Keepers.Subscription.AppendAdjustment(ts.Ctx, idxConsumer.proj, provider1, totalUsageThisIndex, cus.provider1Usage)
		ts.Keepers.Subscription.AppendAdjustment(ts.Ctx, idxConsumer.proj, provider2, totalUsageThisIndex, cus.provider2Usage)
	}

	allAdjustments := ts.Keepers.Subscription.GetAllAdjustment(ts.Ctx)
	require.Len(t, allAdjustments, 8)
	for _, adjustment := range allAdjustments {
		provider, err := ts.Keepers.Subscription.GetProviderFromAdjustment(&adjustment)
		require.NoError(t, err)
		require.True(t, provider == provider1 || provider == provider2)
	}

	consumerAdjustments := ts.Keepers.Subscription.GetConsumerAdjustments(ts.Ctx, consumer1Addr)
	require.Len(t, consumerAdjustments, 4)
	seenProviders := map[string]struct{}{}
	for _, adjustment := range allAdjustments {
		provider, err := ts.Keepers.Subscription.GetProviderFromAdjustment(&adjustment)
		require.NoError(t, err)
		require.True(t, provider == provider1 || provider == provider2)
		seenProviders[provider] = struct{}{}
	}
	require.Len(t, seenProviders, 2)

	providersFactors := ts.Keepers.Subscription.GetAdjustmentFactorProvider(ts.Ctx, consumerAdjustments)
	require.Len(t, providersFactors, 2)
	ts.Keepers.Subscription.RemoveConsumerAdjustments(ts.Ctx, consumer1Addr)
	allAdjustments = ts.Keepers.Subscription.GetAllAdjustment(ts.Ctx)
	require.Len(t, allAdjustments, 4)

	consumerAdjustments = ts.Keepers.Subscription.GetConsumerAdjustments(ts.Ctx, consumer2Addr)
	require.Len(t, consumerAdjustments, 4)
	providersFactors2 := ts.Keepers.Subscription.GetAdjustmentFactorProvider(ts.Ctx, consumerAdjustments)
	require.Len(t, providersFactors2, 2)
	ts.Keepers.Subscription.RemoveConsumerAdjustments(ts.Ctx, consumer2Addr)
	allAdjustments = ts.Keepers.Subscription.GetAllAdjustment(ts.Ctx)
	require.Len(t, allAdjustments, 0)

	// check adjustment values:
	// consuemr 1
	require.True(t, providersFactors[provider1].Equal(math.LegacyMustNewDecFromStr("0.2")), providersFactors[provider1].String())
	require.True(t, providersFactors[provider2].Equal(math.LegacyMustNewDecFromStr("1")), providersFactors[provider2].String())
	// consumer2
	// 120/300 = 0.4, ((100/5) + 0.4 * 100)/201 = 0.29850746, 0.4*300 + 0.29850746*201 = (60+120) / 501 = 0.35928144
	require.True(t, providersFactors2[provider1].Equal(math.LegacyMustNewDecFromStr("180").QuoInt64(501)), providersFactors2[provider1].String())
	// 120/300, 141/201 = 261/501
	require.True(t, providersFactors2[provider2].Equal(math.LegacyMustNewDecFromStr("261").QuoInt64(501)), providersFactors2[provider2].String())
}

func TestAdjustmentEdgeValues(t *testing.T) {
	playbook := []struct {
		name                  string
		firstProviderCU       uint64
		expectedAdjustmentDec string
	}{
		{
			name:                  "all-equal",
			firstProviderCU:       1000,
			expectedAdjustmentDec: "1",
		},
		{
			name:                  "very little",
			firstProviderCU:       1,
			expectedAdjustmentDec: "1",
		},
		{
			name:                  "top",
			firstProviderCU:       1000000000000000000,
			expectedAdjustmentDec: "0.2",
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ts := common.NewTester(t)
			err := ts.SetupForTests("../../../", "LAV1", 1, 1, 1, 5)
			require.NoError(t, err)
			_, consumer1Addr := ts.GetAccount(common.CONSUMER, 0)
			projects := ts.Keepers.Projects.GetAllProjectsForSubscription(ts.Ctx, consumer1Addr)
			require.Len(t, projects, 2)
			providers := ts.Accounts(common.PROVIDER)

			usage := []struct {
				providerUsage []uint64
			}{
				{providerUsage: []uint64{
					play.firstProviderCU, 1000, 1000, 1000, 1000,
				}},
				{providerUsage: []uint64{
					play.firstProviderCU, 1000, 1000, 1000, 1000,
				}},
				{providerUsage: []uint64{
					play.firstProviderCU, 1000, 1000, 1000, 1000,
				}},
				{providerUsage: []uint64{
					play.firstProviderCU, 1000, 1000, 1000, 1000,
				}},
				{providerUsage: []uint64{
					play.firstProviderCU, 1000, 1000, 1000, 1000,
				}},
				{providerUsage: []uint64{
					play.firstProviderCU, 1000, 1000, 1000, 1000,
				}},
				{providerUsage: []uint64{
					play.firstProviderCU, 1000, 1000, 1000, 1000,
				}},
				{providerUsage: []uint64{
					play.firstProviderCU, 1000, 1000, 1000, 1000,
				}},
			}

			totalUsage := func(cus []uint64) uint64 {
				res := uint64(0)
				for _, cu := range cus {
					res += cu
				}
				return res
			}

			for _, usageProviders := range usage {
				totalUsageThisIndex := totalUsage(usageProviders.providerUsage)
				for providerIdx, cu := range usageProviders.providerUsage {
					ts.Keepers.Subscription.AppendAdjustment(ts.Ctx, consumer1Addr, providers[providerIdx].Addr.String(), totalUsageThisIndex, cu)
				}
			}

			consumerAdjustments := ts.Keepers.Subscription.GetConsumerAdjustments(ts.Ctx, consumer1Addr)
			require.NotEmpty(t, consumerAdjustments)
			seenProviders := map[string]struct{}{}
			for _, adjustment := range consumerAdjustments {
				provider, err := ts.Keepers.Subscription.GetProviderFromAdjustment(&adjustment)
				require.NoError(t, err)
				seenProviders[provider] = struct{}{}
			}
			require.Len(t, seenProviders, len(providers))

			providersFactors := ts.Keepers.Subscription.GetAdjustmentFactorProvider(ts.Ctx, consumerAdjustments)
			require.Len(t, providersFactors, len(providers))
			prevFactor := providersFactors[providers[1].Addr.String()]
			for idx, factor := range providersFactors {
				if idx == providers[0].Addr.String() {
					require.True(t, factor.Equal(math.LegacyMustNewDecFromStr(play.expectedAdjustmentDec)), factor.String())
				} else {
					require.Equal(t, prevFactor, factor)
				}
			}
		})
	}
}
