package keeper_test

import (
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/slices"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/rand"
	"github.com/stretchr/testify/require"
)

func (ts *tester) checkProviderUnstaked(provider sdk.AccAddress) {
	_, unstakeStoragefound, _ := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider)
	require.True(ts.T, unstakeStoragefound)
	_, stakeStorageFound, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider)
	require.False(ts.T, stakeStorageFound)
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
	clientsCount := 50
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
	var unresponsiveDataList [][]byte
	for i := 0; i < unresponsiveCount; i++ {
		unresponsiveData, err := json.Marshal([]string{providers[i].Addr.String()})
		require.Nil(t, err)
		unresponsiveDataList = append(unresponsiveDataList, unresponsiveData)
	}

	// create relay requests for that contain complaints about providers
	relayEpoch := ts.BlockHeight()
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		// Get pairing for the client to pick a valid provider
		pairing, err := ts.QueryPairingGetPairing(ts.spec.Name, clients[clientIndex].Addr.String())
		require.Nil(t, err)
		providerIndex := rand.Intn(len(pairing.Providers))
		providerAddress := pairing.Providers[providerIndex].Address
		providerSdkAddress, err := sdk.AccAddressFromBech32(providerAddress)
		require.Nil(t, err)

		cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits*10 + uint64(clientIndex)

		relaySession := ts.newRelaySession(providerAddress, 0, cuSum, relayEpoch, 0)
		relaySession.UnresponsiveProviders = unresponsiveDataList[clientIndex%unresponsiveCount]
		signRelaySession(relaySession, clients[clientIndex].SK)

		relayPaymentMessage := types.MsgRelayPayment{
			Creator: providerAddress,
			Relays:  slices.Slice(relaySession),
		}

		// send relay payment and check the funds did transfer normally
		ts.payAndVerifyBalance(relayPaymentMessage, clients[clientIndex].Addr, providerSdkAddress, true, true)
	}

	// advance enough epochs so the unresponsive providers will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst)

	for i := 0; i < unresponsiveCount; i++ {
		ts.checkProviderUnstaked(providers[i].Addr)
		ts.checkComplainerReset(providers[i].Addr, relayEpoch)
	}

	for i := unresponsiveCount; i < providersCount; i++ {
		ts.checkProviderStaked(providers[i].Addr)
	}
}

// Test to measure the time the check for unresponsiveness every epoch start takes
func TestUnstakingProviderForUnresponsiveness(t *testing.T) {
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
	provider1_balance := ts.GetBalance(provider1_addr)
	staked_amount, _, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider1_addr)
	balanceProvideratBeforeStake := staked_amount.Stake.Amount.Int64() + provider1_balance
	unresponsiveProvidersData, err := json.Marshal([]string{provider1_addr.String()})
	require.Nil(t, err)

	// create relay requests for provider0 that contain complaints about provider1
	relayEpoch := ts.BlockHeight()
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits*10 + uint64(clientIndex)

		relaySession := ts.newRelaySession(provider0_addr.String(), 0, cuSum, relayEpoch, 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		signRelaySession(relaySession, clients[clientIndex].SK)

		relayPaymentMessage := types.MsgRelayPayment{
			Creator: provider0_addr.String(),
			Relays:  slices.Slice(relaySession),
		}

		ts.payAndVerifyBalance(relayPaymentMessage, clients[clientIndex].Addr, provider0_addr, true, true)
	}

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst)

	ts.checkProviderUnstaked(provider1_addr)
	ts.checkComplainerReset(provider1_addr, relayEpoch)
	ts.checkProviderStaked(provider0_addr)

	// advance enough epochs so the current block will be deleted
	epochBlocks := ts.EpochBlocks()
	epochsToSkip := (ts.BlocksToSave() + epochBlocks - 1) / epochBlocks
	ts.AdvanceEpochs(epochsToSkip)

	// validate that the provider is no longer unstaked
	// but also validate that the provider hasn't returned to the stake pool
	_, unstakeStoragefound, _ := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider1_addr)
	require.False(t, unstakeStoragefound)

	_, stakeStorageFound, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider1_addr)
	require.False(t, stakeStorageFound)

	// validate that the provider's balance after the unstake is the same as before he staked
	balanceProviderAfterUnstakeMoneyReturned := ts.GetBalance(provider1_addr)
	require.Equal(t, balanceProvideratBeforeStake, balanceProviderAfterUnstakeMoneyReturned)
}

func TestUnstakingProviderForUnresponsivenessContinueComplainingAfterUnstake(t *testing.T) {
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
	unresponsiveProvidersData, err := json.Marshal([]string{provider1_addr.String()})
	require.Nil(t, err)

	relayEpoch := ts.BlockHeight()
	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	relaySession := ts.newRelaySession(provider0_addr.String(), 0, cuSum, relayEpoch, 0)
	relaySession.UnresponsiveProviders = unresponsiveProvidersData
	signRelaySession(relaySession, clients[0].SK)

	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider0_addr.String(),
		Relays:  slices.Slice(relaySession),
	}

	ts.payAndVerifyBalance(relayPaymentMessage, clients[0].Addr, provider0_addr, true, true)

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst)

	ts.checkProviderUnstaked(provider1_addr)
	ts.checkComplainerReset(provider1_addr, relayEpoch)

	ts.AdvanceEpochs(2)

	// create more relay requests for provider0 that contain complaints about provider1
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider0_addr.String(), 2, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		signRelaySession(relaySession, clients[clientIndex].SK)

		relayPaymentMessage := types.MsgRelayPayment{
			Creator: provider0_addr.String(),
			Relays:  slices.Slice(relaySession),
		}

		ts.payAndVerifyBalance(relayPaymentMessage, clients[clientIndex].Addr, provider0_addr, true, true)
	}

	// test the provider is still unstaked
	_, stakeStorageFound, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider1_addr)
	require.False(t, stakeStorageFound)
	_, unStakeStoragefound, _ := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider1_addr)
	require.True(t, unStakeStoragefound)

	// get the current unstake storage
	storage, foundStorage := ts.Keepers.Epochstorage.GetStakeStorageUnstake(ts.Ctx)
	require.True(t, foundStorage)

	// validate the punished provider is not shown twice (or more) in the unstake storage
	var numberOfAppearances int
	for _, stored := range storage.StakeEntries {
		if stored.Address == provider1_addr.String() {
			numberOfAppearances += 1
		}
	}
	require.Equal(t, numberOfAppearances, 1)
}

func TestNotUnstakingProviderForUnresponsivenessWithMinProviders(t *testing.T) {
	clientsCount := 1
	providersCount := 2

	ts := newTester(t)
	ts.setupForPayments(providersCount, clientsCount, providersCount) // set providers-to-pair
	ts.addProviderGeolocation(2, 2)

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
	unresponsiveProvidersData, err := json.Marshal([]string{provider1_addr.String()})
	require.Nil(t, err)

	// create relay requests for provider0 that contain complaints about provider1
	relayEpoch := ts.BlockHeight()
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits*10 + uint64(clientIndex)

		relaySession := ts.newRelaySession(provider0_addr.String(), 0, cuSum, relayEpoch, 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		signRelaySession(relaySession, clients[clientIndex].SK)

		relayPaymentMessage := types.MsgRelayPayment{
			Creator: provider0_addr.String(),
			Relays:  slices.Slice(relaySession),
		}

		ts.payAndVerifyBalance(relayPaymentMessage, clients[clientIndex].Addr, provider0_addr, true, true)
	}

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}

	ts.AdvanceEpochs(largerConst)

	// test the unresponsive provider1 has been unstaked
	_, unstakeStoragefound, _ := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider1_addr)
	require.False(t, unstakeStoragefound)
	_, stakeStorageFound, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider1_addr)
	require.True(t, stakeStorageFound)
}
