package common

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	conflictconstruct "github.com/lavanet/lava/x/conflict/types/construct"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
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

func CreateResponseConflictMsgDetectionForTest(ctx context.Context, consumer, provider0, provider1 sigs.Account, spec *spectypes.Spec) (detectionMsg *conflicttypes.MsgDetection, reply1, reply2 *pairingtypes.RelayReply, errRet error) {
	detectionMsg = &conflicttypes.MsgDetection{
		Creator: consumer.Addr.String(),
		Conflict: &conflicttypes.MsgDetection_ResponseConflict{
			ResponseConflict: &conflicttypes.ResponseConflict{
				ConflictRelayData0: initConflictRelayData(),
				ConflictRelayData1: initConflictRelayData(),
			},
		},
	}

	responseConflict := detectionMsg.GetResponseConflict()
	// Prepare request and session for provider0.
	prepareRelayData(ctx, responseConflict.ConflictRelayData0, provider0, spec)
	// Sign the session data with the consumer's private key.
	if err := signSessionData(consumer, responseConflict.ConflictRelayData0.Request.RelaySession); err != nil {
		return detectionMsg, nil, nil, err
	}

	// Duplicate the request for provider1 and update provider-specific fields.
	duplicateRequestForProvider(responseConflict, provider1, consumer)
	// Sign the session data with the consumer's private key.
	if err := signSessionData(consumer, responseConflict.ConflictRelayData1.Request.RelaySession); err != nil {
		return detectionMsg, nil, nil, err
	}

	// Create and sign replies for both providers.
	reply1, err := createAndSignReply(provider0, responseConflict.ConflictRelayData0.Request, spec, false)
	if err != nil {
		return detectionMsg, nil, nil, err
	}

	reply2, err = createAndSignReply(provider1, responseConflict.ConflictRelayData1.Request, spec, true)
	if err != nil {
		return detectionMsg, nil, nil, err
	}

	// Construct final conflict relay data with the replies.
	conflictRelayData0, err := finalizeConflictRelayData(consumer, provider0, responseConflict.ConflictRelayData0, reply1)
	if err != nil {
		return detectionMsg, nil, nil, err
	}
	conflictRelayData1, err := finalizeConflictRelayData(consumer, provider1, responseConflict.ConflictRelayData1, reply2)
	if err != nil {
		return detectionMsg, nil, nil, err
	}

	responseConflict.ConflictRelayData0 = conflictRelayData0
	responseConflict.ConflictRelayData1 = conflictRelayData1

	return detectionMsg, reply1, reply2, nil
}

// initConflictRelayData initializes the structure for holding relay conflict data.
func initConflictRelayData() *conflicttypes.ConflictRelayData {
	return &conflicttypes.ConflictRelayData{
		Request: &pairingtypes.RelayRequest{},
		Reply:   &conflicttypes.ReplyMetadata{},
	}
}

// prepareRelayData prepares relay data for a given provider.
func prepareRelayData(ctx context.Context, conflictData *conflicttypes.ConflictRelayData, provider sigs.Account, spec *spectypes.Spec) {
	relayData := &pairingtypes.RelayPrivateData{
		ConnectionType: "",
		ApiUrl:         "",
		Data:           []byte("DUMMYREQUEST"),
		RequestBlock:   100,
		ApiInterface:   "",
		Salt:           []byte{1},
	}

	conflictData.Request.RelayData = relayData
	conflictData.Request.RelaySession = &pairingtypes.RelaySession{
		Provider:    provider.Addr.String(),
		ContentHash: sigs.HashMsg(relayData.GetContentHashData()),
		SessionId:   uint64(1),
		SpecId:      spec.Index,
		CuSum:       0,
		Epoch:       sdk.UnwrapSDKContext(ctx).BlockHeight(),
		RelayNum:    0,
		QosReport:   &pairingtypes.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()},
	}
}

// signSessionData signs the session data with the consumer's private key.
func signSessionData(consumer sigs.Account, relaySession *pairingtypes.RelaySession) error {
	sig, err := sigs.Sign(consumer.SK, *relaySession)
	if err != nil {
		return err
	}

	relaySession.Sig = sig
	return nil
}

// duplicateRequestForProvider duplicates request data for another provider and signs it.
func duplicateRequestForProvider(conflict *conflicttypes.ResponseConflict, provider, consumer sigs.Account) {
	// Clone request data
	temp, _ := conflict.ConflictRelayData0.Request.Marshal()
	conflict.ConflictRelayData1.Request.Unmarshal(temp)

	conflict.ConflictRelayData1.Request.RelaySession.Provider = provider.Addr.String()
	conflict.ConflictRelayData1.Request.RelaySession.Sig = []byte{}
}

// createAndSignReply creates a reply for a provider and signs it.
func createAndSignReply(provider sigs.Account, request *pairingtypes.RelayRequest, spec *spectypes.Spec, addDiffData bool) (*pairingtypes.RelayReply, error) {
	reply := &pairingtypes.RelayReply{
		Data:        []byte("DUMMYREPLY"),
		Sig:         request.RelaySession.Sig,
		LatestBlock: request.RelayData.RequestBlock + int64(spec.BlockDistanceForFinalizedData),
	}

	if addDiffData {
		reply.Data = append(reply.Data, []byte("DIFF")...)
	}

	relayExchange := pairingtypes.NewRelayExchange(*request, *reply)
	sig, err := sigs.Sign(provider.SK, relayExchange)
	if err != nil {
		return reply, err
	}

	reply.Sig = sig

	return reply, nil
}

// finalizeConflictRelayData updates the conflict relay data with the reply information.
func finalizeConflictRelayData(consumer, provider sigs.Account, conflictData *conflicttypes.ConflictRelayData, reply *pairingtypes.RelayReply) (*conflicttypes.ConflictRelayData, error) {
	relayFinalization := conflicttypes.NewRelayFinalizationFromRelaySessionAndRelayReply(conflictData.Request.RelaySession, reply, consumer.Addr)
	sigBlocks, err := sigs.Sign(provider.SK, relayFinalization)
	if err != nil {
		return nil, err
	}

	reply.SigBlocks = sigBlocks

	conflictRelayData := conflictconstruct.ConstructConflictRelayData(reply, conflictData.Request)
	return conflictRelayData, nil
}

func CreateRelayFinalizationForTest(ctx context.Context, consumer, provider sigs.Account, latestBlock int64, finalizationBlockHashes map[int64]string, spec *spectypes.Spec) (*conflicttypes.RelayFinalization, error) {
	relayFinalization := &conflicttypes.RelayFinalization{
		LatestBlock:     latestBlock,
		ConsumerAddress: consumer.Addr.String(),
	}

	conflictData := initConflictRelayData()
	prepareRelayData(ctx, conflictData, provider, spec)
	if err := signSessionData(consumer, conflictData.Request.RelaySession); err != nil {
		return relayFinalization, err
	}

	relayFinalization.RelaySession = conflictData.Request.RelaySession

	jsonStr, err := json.Marshal(finalizationBlockHashes)
	if err != nil {
		return relayFinalization, err
	}

	relayFinalization.FinalizedBlocksHashes = jsonStr

	// Sign relay reply for provider
	sig0, err := sigs.Sign(provider.SK, relayFinalization)
	if err != nil {
		return relayFinalization, err
	}
	relayFinalization.SigBlocks = sig0

	return relayFinalization, nil
}
