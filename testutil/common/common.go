package common

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	testkeeper "github.com/lavanet/lava/v5/testutil/keeper"
	"github.com/lavanet/lava/v5/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/v5/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func CreateNewAccount(ctx context.Context, keepers testkeeper.Keepers, balance int64) (acc sigs.Account) {
	acc = sigs.GenerateDeterministicFloatingKey(testkeeper.Randomizer)
	testkeeper.Randomizer.Inc()
	coins := sdk.NewCoins(sdk.NewCoin(keepers.StakingKeeper.BondDenom(sdk.UnwrapSDKContext(ctx)), sdk.NewInt(balance)))
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), acc.Addr, coins)
	return
}

func StakeAccount(t *testing.T, ctx context.Context, keepers testkeeper.Keepers, servers testkeeper.Servers, acc sigs.Account, spec spectypes.Spec, stake int64, validator sigs.Account) {
	endpoints := []epochstoragetypes.Endpoint{}
	for _, collection := range spec.ApiCollections {
		endpoints = append(endpoints, epochstoragetypes.Endpoint{IPPORT: "123", ApiInterfaces: []string{collection.CollectionData.ApiInterface}, Geolocation: 1})
	}

	_, err := servers.PairingServer.StakeProvider(ctx, &pairingtypes.MsgStakeProvider{
		Creator:            acc.Addr.String(),
		Address:            acc.Addr.String(),
		ChainID:            spec.Index,
		Amount:             sdk.NewCoin(keepers.StakingKeeper.BondDenom(sdk.UnwrapSDKContext(ctx)), sdk.NewInt(stake)),
		Geolocation:        1,
		Endpoints:          endpoints,
		DelegateLimit:      sdk.NewCoin(keepers.StakingKeeper.BondDenom(sdk.UnwrapSDKContext(ctx)), sdk.ZeroInt()),
		DelegateCommission: 100,
		Validator:          sdk.ValAddress(validator.Addr).String(),
		Description:        MockDescription(),
	})
	require.NoError(t, err)
}

func BuySubscription(ctx context.Context, keepers testkeeper.Keepers, servers testkeeper.Servers, acc sigs.Account, plan string) {
	servers.SubscriptionServer.Buy(ctx, &subscriptiontypes.MsgBuy{Creator: acc.Addr.String(), Consumer: acc.Addr.String(), Index: plan, Duration: 1})
}

func MockDescription() stakingtypes.Description {
	return stakingtypes.NewDescription("prov", "iden", "web", "sec", "details")
}

func BuildRelayRequest(ctx context.Context, provider string, contentHash []byte, cuSum uint64, spec string, qos *pairingtypes.QualityOfServiceReport) *pairingtypes.RelaySession {
	return BuildRelayRequestWithBadge(ctx, provider, contentHash, uint64(1), cuSum, spec, qos, nil)
}

func BuildRelayRequestWithSession(ctx context.Context, provider string, contentHash []byte, sessionId uint64, cuSum uint64, spec string, qos *pairingtypes.QualityOfServiceReport) *pairingtypes.RelaySession {
	return BuildRelayRequestWithBadge(ctx, provider, contentHash, sessionId, cuSum, spec, qos, nil)
}

func BuildRelayRequestWithBadge(ctx context.Context, provider string, contentHash []byte, sessionId uint64, cuSum uint64, spec string, qos *pairingtypes.QualityOfServiceReport, badge *pairingtypes.Badge) *pairingtypes.RelaySession {
	relaySession := &pairingtypes.RelaySession{
		Provider:    provider,
		ContentHash: contentHash,
		SessionId:   sessionId,
		SpecId:      spec,
		CuSum:       cuSum,
		Epoch:       sdk.UnwrapSDKContext(ctx).BlockHeight(),
		RelayNum:    0,
		QosReport:   qos,
		LavaChainId: sdk.UnwrapSDKContext(ctx).BlockHeader().ChainID,
		Badge:       badge,
	}
	if qos != nil {
		qos.ComputeQoS()
	}
	return relaySession
}
