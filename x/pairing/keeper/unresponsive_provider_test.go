package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/rand"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func (ts *tester) checkProviderFreeze(provider sdk.AccAddress, shouldFreeze bool) {
	stakeEntry, stakeStorageFound, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider)
	require.True(ts.T, stakeStorageFound)
	if shouldFreeze {
		require.Equal(ts.T, uint64(epochstoragetypes.FROZEN_BLOCK), stakeEntry.StakeAppliedBlock)
	} else {
		require.NotEqual(ts.T, uint64(epochstoragetypes.FROZEN_BLOCK), stakeEntry.StakeAppliedBlock)
	}
}

func (ts *tester) checkComplainerReset(provider sdk.AccAddress, epoch uint64) {
	// validate the complainers CU field in the unresponsive provider's providerPaymentStorage
	// was reset after being punished (use the epoch from the relay - when it got reported)
	providerPaymentStorageKey := ts.Keepers.Pairing.GetProviderPaymentStorageKey(ts.Ctx, ts.spec.Name, epoch, provider)
	providerPaymentStorage, found := ts.Keepers.Pairing.GetProviderPaymentStorage(ts.Ctx, providerPaymentStorageKey)
	require.Equal(ts.T, true, found)
	require.Equal(ts.T, uint64(0), providerPaymentStorage.ComplainersTotalCu)
}

func (ts *tester) checkProviderStaked(provider sdk.AccAddress) {
	_, unstakeStoragefound, _ := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider)
	require.False(ts.T, unstakeStoragefound)
	_, stakeStorageFound, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider)
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

	ts.AdvanceEpochs(largerConst)

	for i := 0; i < unresponsiveCount; i++ {
		ts.checkProviderFreeze(providers[i].Addr, true)
		ts.checkComplainerReset(providers[i].Addr, relayEpoch)
	}

	for i := unresponsiveCount; i < providersCount; i++ {
		ts.checkProviderStaked(providers[i].Addr)
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
	provider0_addr := sdk.MustAccAddressFromBech32(pairing.Providers[0].Address)
	provider1_addr := sdk.MustAccAddressFromBech32(pairing.Providers[1].Address)

	// get provider1's balance before the stake
	unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider1_addr.String()}}

	// create relay requests for provider0 that contain complaints about provider1
	relayEpoch := ts.BlockHeight()
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits*10 + uint64(clientIndex)

		relaySession := ts.newRelaySession(provider0_addr.String(), 0, cuSum, relayEpoch, 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		relayPaymentMessage := types.MsgRelayPayment{
			Creator: provider0_addr.String(),
			Relays:  lavaslices.Slice(relaySession),
		}

		ts.relayPaymentWithoutPay(relayPaymentMessage, true)
	}

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst)

	ts.checkProviderFreeze(provider1_addr, true)
	ts.checkComplainerReset(provider1_addr, relayEpoch)
	ts.checkProviderStaked(provider0_addr)
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
	provider0_addr := sdk.MustAccAddressFromBech32(pairing.Providers[0].Address)
	provider1_addr := sdk.MustAccAddressFromBech32(pairing.Providers[1].Address)

	// create relay requests for provider0 that contain complaints about provider1
	unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider1_addr.String()}}

	relayEpoch := ts.BlockHeight()
	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	relaySession := ts.newRelaySession(provider0_addr.String(), 0, cuSum, relayEpoch, 0)

	relaySession.UnresponsiveProviders = unresponsiveProvidersData
	sig, err := sigs.Sign(clients[0].SK, *relaySession)
	relaySession.Sig = sig
	require.NoError(t, err)
	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider0_addr.String(),
		Relays:  lavaslices.Slice(relaySession),
	}

	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst)

	ts.checkProviderFreeze(provider1_addr, true)
	ts.checkComplainerReset(provider1_addr, relayEpoch)

	ts.AdvanceEpochs(2)

	// create more relay requests for provider0 that contain complaints about provider1
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider0_addr.String(), 2, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)

		relayPaymentMessage := types.MsgRelayPayment{
			Creator: provider0_addr.String(),
			Relays:  lavaslices.Slice(relaySession),
		}

		ts.relayPaymentWithoutPay(relayPaymentMessage, true)
	}

	// test the provider is still frozen
	ts.checkProviderFreeze(provider1_addr, true)
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
		ts.AdvanceEpochs(largerConst + recommendedEpochNumToCollectPayment)

		// find two providers in the pairing
		pairing, err := ts.QueryPairingGetPairing(ts.spec.Name, clients[0].Addr.String())
		require.NoError(t, err)
		provider0_addr := sdk.MustAccAddressFromBech32(pairing.Providers[0].Address)
		provider1_addr := sdk.MustAccAddressFromBech32(pairing.Providers[1].Address)

		// create unresponsive data that includes provider1 being unresponsive
		unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider1_addr.String()}}
		// create relay requests for provider0 that contain complaints about provider1
		relayEpoch := ts.BlockHeight()
		for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
			cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits*10 + uint64(clientIndex)

			relaySession := ts.newRelaySession(provider0_addr.String(), 0, cuSum, relayEpoch, 0)
			relaySession.UnresponsiveProviders = unresponsiveProvidersData
			sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
			relaySession.Sig = sig
			require.NoError(t, err)

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: provider0_addr.String(),
				Relays:  lavaslices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(relayPaymentMessage, clients[clientIndex].Addr, provider0_addr, true, true, 100)
		}

		// advance enough epochs so the unresponsive provider will be punished
		if largerConst < recommendedEpochNumToCollectPayment {
			largerConst = recommendedEpochNumToCollectPayment
		}

		ts.AdvanceEpochs(largerConst)

		// test the unresponsive provider1 hasn't froze
		ts.checkProviderFreeze(provider1_addr, play.shouldBeFrozen)
	}
}
