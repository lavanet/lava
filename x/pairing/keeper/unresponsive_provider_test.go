package keeper_test

import (
	"testing"
	"time"

	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	"github.com/lavanet/lava/v2/utils/rand"
	"github.com/lavanet/lava/v2/utils/sigs"
	"github.com/lavanet/lava/v2/x/pairing/keeper"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func (ts *tester) checkProviderJailed(provider string, shouldFreeze bool) {
	stakeEntry, stakeStorageFound := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider)
	require.True(ts.T, stakeStorageFound)
	require.Equal(ts.T, shouldFreeze, stakeEntry.IsJailed(ts.Ctx.BlockTime().UTC().Unix()))
	if shouldFreeze {
		jailDelta := keeper.SOFT_JAIL_TIME
		if stakeEntry.Jails > keeper.SOFT_JAILS {
			jailDelta = keeper.HARD_JAIL_TIME
		}
		require.InDelta(ts.T, stakeEntry.JailEndTime, ts.BlockTime().UTC().Unix(), float64(jailDelta))
	}
}

func (ts *tester) checkComplainerReset(provider string, epoch uint64) {
	// validate the complainers CU field in the unresponsive provider's providerPaymentStorage
	// was reset after being punished (use the epoch from the relay - when it got reported)
	_, found := ts.Keepers.Pairing.GetProviderEpochComplainerCu(ts.Ctx, epoch, provider, ts.spec.Name)
	require.Equal(ts.T, false, found)
}

func (ts *tester) checkProviderStaked(provider string) {
	_, unstakeStoragefound := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider)
	require.False(ts.T, unstakeStoragefound)
	_, stakeStorageFound := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider)
	require.True(ts.T, stakeStorageFound)
}

func TestUnresponsivenessStressTest(t *testing.T) {
	rand.InitRandomSeed()
	clientsCount := 100
	providersCount := 6

	ts := newTester(t)
	ts.setupForPayments(providersCount, clientsCount, providersCount-1) // set providers-to-pair

	clients := ts.Accounts(common.CONSUMER)
	providers := ts.Accounts(common.PROVIDER)

	recommendedEpochNumToCollectPayment := ts.Keepers.Pairing.RecommendedEpochNumToCollectPayment(ts.Ctx)

	largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
	if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
		largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
	}

	// advance enough epochs so we can check punishment due to unresponsiveness
	// (if the epoch is too early, there's no punishment)
	ts.AdvanceEpochs(largerConst + recommendedEpochNumToCollectPayment)

	// create list of providers to claim unresponsiveness about
	unresponsiveCount := 1
	var unresponsiveDataList [][]*types.ReportedProvider
	for i := 0; i < unresponsiveCount; i++ {
		unresponsiveData := []*types.ReportedProvider{{Address: providers[i].Addr.String()}}
		unresponsiveDataList = append(unresponsiveDataList, unresponsiveData)
	}

	// create relay requests for that contain complaints about providers
	relayEpoch := ts.BlockHeight()
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		// Get pairing for the client to pick a valid provider
		pairing, err := ts.QueryPairingGetPairing(ts.spec.Name, clients[clientIndex].Addr.String())
		require.NoError(t, err)
		providerIndex := rand.Intn(len(pairing.Providers))
		providerAddress := pairing.Providers[providerIndex].Address
		// NOTE: the following loop contains a random factor in it. We make sure that we pick
		// a provider in random that is not one of the defined unresponsive providers
		// If we did, we pick one of the providers in random again and check whether it's
		// one of the unresponsive ones.
		// With a large number of provider and a small number of unresponsive providers,
		// this is very likely not to get stuck in an infinite loop
		for {
			isProviderUnresponsive := false
			for _, unresponsiveProviderList := range unresponsiveDataList {
				for _, unresponsiveProvider := range unresponsiveProviderList {
					if providerAddress == unresponsiveProvider.Address {
						isProviderUnresponsive = true
					}
				}
			}
			if !isProviderUnresponsive {
				break
			}
			providerIndex = rand.Intn(len(pairing.Providers))
			providerAddress = pairing.Providers[providerIndex].Address
		}

		cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits*10 + uint64(clientIndex)

		relaySession := ts.newRelaySession(providerAddress, 0, cuSum, relayEpoch, 0)
		relaySession.UnresponsiveProviders = unresponsiveDataList[clientIndex%unresponsiveCount]
		sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		relayPaymentMessage := types.MsgRelayPayment{
			Creator: providerAddress,
			Relays:  lavaslices.Slice(relaySession),
		}

		// send relay payment and check the funds did transfer normally
		ts.relayPaymentWithoutPay(relayPaymentMessage, true)
	}

	// advance enough epochs so the unresponsive providers will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst, time.Nanosecond)

	for i := 0; i < unresponsiveCount; i++ {
		ts.checkProviderJailed(providers[i].Addr.String(), true)
		ts.checkComplainerReset(providers[i].Addr.String(), relayEpoch)
	}

	for i := unresponsiveCount; i < providersCount; i++ {
		ts.checkProviderStaked(providers[i].Addr.String())
	}
}

// Test to measure the time the check for unresponsiveness every epoch start takes
func TestFreezingProviderForUnresponsiveness(t *testing.T) {
	// setup test for unresponsiveness
	clientsCount := 1
	providersCount := 10

	ts := newTester(t)
	ts.setupForPayments(providersCount, clientsCount, providersCount-1) // set providers-to-pair

	clients := ts.Accounts(common.CONSUMER)

	recommendedEpochNumToCollectPayment := ts.Keepers.Pairing.RecommendedEpochNumToCollectPayment(ts.Ctx)

	largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
	if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
		largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
	}

	// advance enough epochs so we can check punishment due to unresponsiveness
	// (if the epoch is too early, there's no punishment)
	ts.AdvanceEpochs(largerConst + recommendedEpochNumToCollectPayment)

	// find two providers in the pairing
	pairing, err := ts.QueryPairingGetPairing(ts.spec.Name, clients[0].Addr.String())
	require.NoError(t, err)
	provider0 := pairing.Providers[0].Address
	provider1 := pairing.Providers[1].Address

	// get provider1's balance before the stake
	unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider1}}

	// create relay requests for provider0 that contain complaints about provider1
	relayEpoch := ts.BlockHeight()
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits*10 + uint64(clientIndex)

		relaySession := ts.newRelaySession(provider0, 0, cuSum, relayEpoch, 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		relayPaymentMessage := types.MsgRelayPayment{
			Creator: provider0,
			Relays:  lavaslices.Slice(relaySession),
		}

		ts.relayPaymentWithoutPay(relayPaymentMessage, true)
	}

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst, time.Second)

	ts.checkProviderJailed(provider1, true)
	ts.checkComplainerReset(provider1, relayEpoch)
	ts.checkProviderStaked(provider0)
}

func TestFreezingProviderForUnresponsivenessContinueComplainingAfterFreeze(t *testing.T) {
	clientsCount := 1
	providersCount := 5

	ts := newTester(t)
	ts.setupForPayments(providersCount, clientsCount, providersCount-1) // set providers-to-pair

	clients := ts.Accounts(common.CONSUMER)

	recommendedEpochNumToCollectPayment := ts.Keepers.Pairing.RecommendedEpochNumToCollectPayment(ts.Ctx)

	largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
	if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
		largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
	}

	// advance enough epochs so we can check punishment due to unresponsiveness
	// (if the epoch is too early, there's no punishment)
	ts.AdvanceEpochs(largerConst + recommendedEpochNumToCollectPayment)

	// find two providers in the pairing
	pairing, err := ts.QueryPairingGetPairing(ts.spec.Name, clients[0].Addr.String())
	require.NoError(t, err)
	provider0 := pairing.Providers[0].Address
	provider1 := pairing.Providers[1].Address

	// create relay requests for provider0 that contain complaints about provider1
	unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider1}}

	relayEpoch := ts.BlockHeight()
	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	relaySession := ts.newRelaySession(provider0, 0, cuSum, relayEpoch, 0)

	relaySession.UnresponsiveProviders = unresponsiveProvidersData
	sig, err := sigs.Sign(clients[0].SK, *relaySession)
	relaySession.Sig = sig
	require.NoError(t, err)
	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider0,
		Relays:  lavaslices.Slice(relaySession),
	}

	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst, time.Second)

	ts.checkProviderJailed(provider1, true)
	ts.checkComplainerReset(provider1, relayEpoch)

	ts.AdvanceEpochs(2, time.Second)

	// look for the first provider that is not provider1
	pairing, err = ts.QueryPairingGetPairing(ts.spec.Name, clients[0].Addr.String())
	require.NoError(t, err)
	for _, provider := range pairing.Providers {
		if provider.Address != provider1 {
			provider0 = provider.Address
			break
		}
	}

	// create more relay requests for provider0 that contain complaints about provider1
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider0, 2, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)

		relayPaymentMessage := types.MsgRelayPayment{
			Creator: provider0,
			Relays:  lavaslices.Slice(relaySession),
		}

		ts.relayPaymentWithoutPay(relayPaymentMessage, true)
	}

	// test the provider is still frozen
	ts.checkProviderJailed(provider1, true)
}

func TestNotFreezingProviderForUnresponsivenessWithMinProviders(t *testing.T) {
	clientsCount := 1
	providersCount := 2
	plays := []struct {
		providersToPair uint64
		shouldBeFrozen  bool
	}{
		{
			providersToPair: 4,
			shouldBeFrozen:  false,
		},
		{
			providersToPair: 2,
			shouldBeFrozen:  true,
		},
	}
	for _, play := range plays {
		ts := newTester(t)
		ts.setupForPayments(providersCount, clientsCount, providersCount) // set providers-to-pair
		err := ts.addProviderGeolocation(2, 2)
		require.NoError(t, err)
		plan := ts.plan
		plan.PlanPolicy.MaxProvidersToPair = play.providersToPair
		ts.ModifyPlan(ts.plan.Index, plan)
		clients := ts.Accounts(common.CONSUMER)
		recommendedEpochNumToCollectPayment := ts.Keepers.Pairing.RecommendedEpochNumToCollectPayment(ts.Ctx)

		// check which const is larger
		largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
		if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
			largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
		}
		// advance enough epochs so we can check punishment due to unresponsiveness
		// (if the epoch is too early, there's no punishment)
		ts.AdvanceEpochs(largerConst+recommendedEpochNumToCollectPayment, time.Nanosecond)

		// find two providers in the pairing
		pairing, err := ts.QueryPairingGetPairing(ts.spec.Name, clients[0].Addr.String())
		require.NoError(t, err)
		provider0Provider := pairing.Providers[0].Address
		provider1Provider := pairing.Providers[1].Address

		// create unresponsive data that includes provider1 being unresponsive
		unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider1Provider}}
		// create relay requests for provider0 that contain complaints about provider1
		relayEpoch := ts.BlockHeight()
		for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
			cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits*10 + uint64(clientIndex)

			relaySession := ts.newRelaySession(provider0Provider, 0, cuSum, relayEpoch, 0)
			relaySession.UnresponsiveProviders = unresponsiveProvidersData
			sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
			relaySession.Sig = sig
			require.NoError(t, err)

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: provider0Provider,
				Relays:  lavaslices.Slice(relaySession),
			}

			ts.relayPaymentWithoutPay(relayPaymentMessage, true)
		}

		// advance enough epochs so the unresponsive provider will be punished
		if largerConst < recommendedEpochNumToCollectPayment {
			largerConst = recommendedEpochNumToCollectPayment
		}

		ts.AdvanceEpochs(largerConst, time.Nanosecond)

		// test the unresponsive provider1 hasn't froze
		ts.checkProviderJailed(provider1Provider, play.shouldBeFrozen)
	}
}

// Test to measure the time the check for unresponsiveness every epoch start takes
func TestJailProviderForUnresponsiveness(t *testing.T) {
	// setup test for unresponsiveness
	clientsCount := 1
	providersCount := 10

	ts := newTester(t)
	ts.setupForPayments(providersCount, clientsCount, providersCount-1) // set providers-to-pair

	clients := ts.Accounts(common.CONSUMER)

	recommendedEpochNumToCollectPayment := ts.Keepers.Pairing.RecommendedEpochNumToCollectPayment(ts.Ctx)

	largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
	if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
		largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
	}

	// advance enough epochs so we can check punishment due to unresponsiveness
	// (if the epoch is too early, there's no punishment)
	ts.AdvanceEpochs(largerConst + recommendedEpochNumToCollectPayment)

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	// find two providers in the pairing
	pairing, err := ts.QueryPairingGetPairing(ts.spec.Name, clients[0].Addr.String())
	require.NoError(t, err)
	provider0 := pairing.Providers[0].Address
	provider1 := pairing.Providers[1].Address

	jailProvider := func() {
		found := false
		// make sure our provider is in the pairing
		for !found {
			ts.AdvanceEpoch(0)
			pairing1, err := ts.QueryPairingVerifyPairing(ts.spec.Name, clients[0].Addr.String(), provider0, ts.BlockHeight())
			require.NoError(t, err)

			pairing2, err := ts.QueryPairingVerifyPairing(ts.spec.Name, clients[0].Addr.String(), provider1, ts.BlockHeight())
			require.NoError(t, err)
			found = pairing1.Valid && pairing2.Valid
		}

		// create relay requests for provider0 that contain complaints about provider1
		unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider1}}
		relayEpoch := ts.BlockHeight()
		cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

		relaySession := ts.newRelaySession(provider0, 0, cuSum, relayEpoch, 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		sig, err := sigs.Sign(clients[0].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		relayPaymentMessage := types.MsgRelayPayment{
			Creator: provider0,
			Relays:  lavaslices.Slice(relaySession),
		}
		ts.relayPaymentWithoutPay(relayPaymentMessage, true)

		ts.AdvanceEpochs(recommendedEpochNumToCollectPayment+1, 0)
		ts.checkProviderJailed(provider1, true)
		ts.checkComplainerReset(provider1, relayEpoch)
		ts.checkProviderStaked(provider0)

		res, err := ts.QueryPairingVerifyPairing(ts.spec.Name, clients[0].Addr.String(), provider1, ts.BlockHeight())
		require.NoError(t, err)
		require.False(t, res.Valid)
	}

	// jail first time
	jailProvider()
	// try to unfreeze with increase of self delegation
	_, err = ts.TxDualstakingDelegate(provider1, provider1, ts.spec.Index, common.NewCoin(ts.TokenDenom(), 1))
	require.Nil(t, err)

	_, err = ts.TxPairingUnfreezeProvider(provider1, ts.spec.Index)
	require.Nil(t, err)

	ts.checkProviderJailed(provider1, true)

	// advance epoch and one hour to leave jail
	ts.AdvanceBlock(time.Hour)
	ts.AdvanceEpoch(0)
	ts.checkProviderJailed(provider1, false)

	// jail second time
	jailProvider()
	_, err = ts.TxPairingUnfreezeProvider(provider1, ts.spec.Index)
	require.Nil(t, err)

	ts.checkProviderJailed(provider1, true)

	// advance epoch and one hour to leave jail
	ts.AdvanceBlock(time.Hour)
	ts.AdvanceEpoch(0)
	ts.checkProviderJailed(provider1, false)

	// jail third time
	jailProvider()
	// try to unfreeze with increase of self delegation
	_, err = ts.TxDualstakingDelegate(provider1, provider1, ts.spec.Index, common.NewCoin(ts.TokenDenom(), 1))
	require.Nil(t, err)

	_, err = ts.TxPairingUnfreezeProvider(provider1, ts.spec.Index)
	require.NotNil(t, err)

	// advance epoch and one hour to try leave jail
	ts.AdvanceBlock(time.Hour)
	ts.AdvanceEpoch(0)
	ts.checkProviderJailed(provider1, true)

	// advance epoch and 24 hour to try leave jail
	ts.AdvanceBlock(24 * time.Hour)
	ts.AdvanceEpoch(0)
	ts.checkProviderJailed(provider1, false)

	_, err = ts.TxPairingUnfreezeProvider(provider1, ts.spec.Index)
	require.Nil(t, err)
	ts.AdvanceEpochs(largerConst + recommendedEpochNumToCollectPayment)

	// jail first time again
	jailProvider()
	_, err = ts.TxPairingUnfreezeProvider(provider1, ts.spec.Index)
	require.Nil(t, err)

	ts.checkProviderJailed(provider1, true)

	// advance epoch and one hour to leave jail
	ts.AdvanceBlock(time.Hour)
	ts.AdvanceEpoch(0)
	ts.checkProviderJailed(provider1, false)
}
