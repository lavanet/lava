package chainlib

import (
	"context"
	"sync"
	"testing"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/provideroptimizer"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/rand"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestConsumerWSSubscriptionManager(t *testing.T) {
	// This test does the following:
	// 1. Create a new ConsumerWSSubscriptionManager
	// 2. Start a new subscription for the first time -> should call SendParsedRelay once
	// 3. Start a subscription again, same params, same dappKey -> should not call SendParsedRelay
	// 4. Start a subscription again, same params, different dappKey -> should not call SendParsedRelay
	// 5. Start a new subscription, different params, same dappKey -> should call SendParsedRelay
	// 6. Start a subscription again, different params, different dappKey -> should call SendParsedRelay
	// 7. Unsubscribe from the first subscription -> should call CancelSubscriptionContext and SendParsedRelay
	// 8. Unsubscribe from the second subscription -> should call CancelSubscriptionContext and SendParsedRelay

	playbook := []struct {
		name                     string
		specId                   string
		apiInterface             string
		connectionType           string
		subscriptionRequestData1 []byte
		subscriptionFirstReply1  []byte
		subscriptionRequestData2 []byte
		subscriptionFirstReply2  []byte
	}{
		{
			name:                     "TendermintRPC",
			specId:                   "LAV1",
			apiInterface:             spectypes.APIInterfaceTendermintRPC,
			connectionType:           "",
			subscriptionRequestData1: []byte(`{"jsonrpc":"2.0","id":3,"method":"subscribe","params":{"query":"tm.event='NewBlock'"}}`),
			subscriptionFirstReply1:  []byte(`{"jsonrpc":"2.0","id":3,"result":{}}`),
			subscriptionRequestData2: []byte(`{"jsonrpc":"2.0","id":4,"method":"subscribe","params":{"query":"tm.event= 'NewBlock'"}}`),
			subscriptionFirstReply2:  []byte(`{"jsonrpc":"2.0","id":4,"result":{}}`),
		},
		{
			name:                     "JsonRPC",
			specId:                   "ETH1",
			apiInterface:             spectypes.APIInterfaceJsonRPC,
			connectionType:           "POST",
			subscriptionRequestData1: []byte(`{"jsonrpc":"2.0","id":5,"method":"eth_subscribe","params":["newHeads"]}`),
			subscriptionFirstReply1:  []byte(`{"jsonrpc":"2.0","id":5,"result":["0x1234567890"]}`),
			subscriptionRequestData2: []byte(`{"jsonrpc":"2.0","id":6,"method":"eth_subscribe","params":["logs"]}`),
			subscriptionFirstReply2:  []byte(`{"jsonrpc":"2.0","id":6,"result":["0x2134567890"]}`),
		},
	}

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ts := SetupForTests(t, 1, play.specId, "../../")

			dapp1 := "dapp1"
			dapp2 := "dapp2"

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			chainParser, _, _, _, _, err := CreateChainLibMocks(ts.Ctx, play.specId, play.apiInterface, nil, nil, "../../", nil)
			require.NoError(t, err)

			chainMessage1, err := chainParser.ParseMsg("", play.subscriptionRequestData1, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			relaySender := NewMockRelaySender(ctrl)
			relaySender.
				EXPECT().
				CreateDappKey(gomock.Any(), gomock.Any()).
				DoAndReturn(func(dappID, consumerIp string) string {
					return dappID + consumerIp
				}).
				AnyTimes()
			relaySender.
				EXPECT().
				SetConsistencySeenBlock(gomock.Any(), gomock.Any()).
				AnyTimes()

			relaySender.EXPECT()

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(chainMessage1, nil, nil, nil).
				AnyTimes()

			mockRelayerClient1 := pairingtypes.NewMockRelayer_RelaySubscribeClient(ctrl)

			relayResult1 := &common.RelayResult{
				ReplyServer: mockRelayerClient1,
				ProviderInfo: common.ProviderInfo{
					ProviderAddress: ts.Providers[0].Addr.String(),
				},
				Reply: &pairingtypes.RelayReply{
					Data:        play.subscriptionFirstReply1,
					LatestBlock: 1,
				},
				Request: &pairingtypes.RelayRequest{
					RelayData: &pairingtypes.RelayPrivateData{
						Data: play.subscriptionRequestData1,
					},
					RelaySession: &pairingtypes.RelaySession{},
				},
			}

			relayResult1.Reply, err = lavaprotocol.SignRelayResponse(ts.Consumer.Addr, *relayResult1.Request, ts.Providers[0].SK, relayResult1.Reply, true)
			require.NoError(t, err)

			mockRelayerClient1.
				EXPECT().
				Context().
				Return(context.Background()).
				AnyTimes()

			mockRelayerClient1.
				EXPECT().
				RecvMsg(gomock.Any()).
				DoAndReturn(func(msg interface{}) error {
					relayReply, ok := msg.(*pairingtypes.RelayReply)
					require.True(t, ok)

					*relayReply = *relayResult1.Reply
					return nil
				}).
				AnyTimes()

			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(1) // Should call SendParsedRelay, because it is the first time we subscribe

			consumerSessionManager := CreateConsumerSessionManager(play.specId, play.apiInterface, ts.Consumer.Addr.String())

			unsubscribeParamsExtractor := func(request ChainMessage, reply *rpcclient.JsonrpcMessage) string {
				return ""
			}

			// Create a new ConsumerWSSubscriptionManager
			manager := NewConsumerWSSubscriptionManager(consumerSessionManager, relaySender, nil, play.connectionType, chainParser, lavasession.NewActiveSubscriptionProvidersStorage(), unsubscribeParamsExtractor)

			// Start a new subscription for the first time, called SendParsedRelay once
			ctx := utils.WithUniqueIdentifier(ts.Ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan, err := manager.StartSubscription(ctx, chainMessage1, nil, nil, dapp1, ts.Consumer.Addr.String(), nil)
			assert.NoError(t, err)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.NotNil(t, repliesChan)

			go func() {
				for subMsg := range repliesChan {
					require.Equal(t, string(play.subscriptionFirstReply1), string(subMsg.Data))
				}
			}()

			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(0) // Should not call SendParsedRelay, because it is already subscribed

			// Start a subscription again, same params, same dappKey, should not call SendParsedRelay
			ctx = utils.WithUniqueIdentifier(ts.Ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan, err = manager.StartSubscription(ctx, chainMessage1, nil, nil, dapp1, ts.Consumer.Addr.String(), nil)
			assert.NoError(t, err)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.Nil(t, repliesChan) // Same subscription, same dappKey, no need for a new channel

			// Start a subscription again, same params, different dappKey, should not call SendParsedRelay
			ctx = utils.WithUniqueIdentifier(ts.Ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan, err = manager.StartSubscription(ctx, chainMessage1, nil, nil, dapp2, ts.Consumer.Addr.String(), nil)
			assert.NoError(t, err)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.NotNil(t, repliesChan) // Same subscription, but different dappKey, so will create new channel

			go func() {
				for subMsg := range repliesChan {
					require.Equal(t, string(play.subscriptionFirstReply1), string(subMsg.Data))
				}
			}()

			// Prepare for the next subscription
			chainMessage2, err := chainParser.ParseMsg("", play.subscriptionRequestData2, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(chainMessage2, nil, nil, nil).
				AnyTimes()

			mockRelayerClient2 := pairingtypes.NewMockRelayer_RelaySubscribeClient(ctrl)
			mockRelayerClient2.
				EXPECT().
				Context().
				Return(context.Background()).
				AnyTimes()

			relayResult2 := &common.RelayResult{
				ReplyServer: mockRelayerClient2,
				ProviderInfo: common.ProviderInfo{
					ProviderAddress: ts.Providers[0].Addr.String(),
				},
				Reply: &pairingtypes.RelayReply{
					Data: play.subscriptionFirstReply2,
				},
				Request: &pairingtypes.RelayRequest{
					RelayData: &pairingtypes.RelayPrivateData{
						Data: play.subscriptionRequestData2,
					},
					RelaySession: &pairingtypes.RelaySession{},
				},
			}

			relayResult2.Reply, err = lavaprotocol.SignRelayResponse(ts.Consumer.Addr, *relayResult2.Request, ts.Providers[0].SK, relayResult2.Reply, true)
			require.NoError(t, err)

			mockRelayerClient2.
				EXPECT().
				RecvMsg(gomock.Any()).
				DoAndReturn(func(msg interface{}) error {
					relayReply, ok := msg.(*pairingtypes.RelayReply)
					require.True(t, ok)

					*relayReply = *relayResult2.Reply
					return nil
				}).
				AnyTimes()

			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult2, nil).
				Times(1) // Should call SendParsedRelay, because it is the first time we subscribe

			// Start a subscription again, different params, same dappKey, should call SendParsedRelay
			ctx = utils.WithUniqueIdentifier(ts.Ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan, err = manager.StartSubscription(ctx, chainMessage2, nil, nil, dapp1, ts.Consumer.Addr.String(), nil)
			assert.NoError(t, err)
			assert.Equal(t, string(play.subscriptionFirstReply2), string(firstReply.Data))
			assert.NotNil(t, repliesChan) // New subscription, new channel

			go func() {
				for subMsg := range repliesChan {
					require.Equal(t, string(play.subscriptionFirstReply2), string(subMsg.Data))
				}
			}()

			// Prepare for unsubscribe from the first subscription
			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(0) // Should call SendParsedRelay, because it unsubscribed

			ctx = utils.WithUniqueIdentifier(ts.Ctx, utils.GenerateUniqueIdentifier())
			err = manager.Unsubscribe(ctx, chainMessage1, nil, relayResult1.Request.RelayData, dapp2, ts.Consumer.Addr.String(), nil)
			require.NoError(t, err)
			wg := sync.WaitGroup{}
			wg.Add(2)

			relaySender.
				EXPECT().
				CancelSubscriptionContext(gomock.Any()).
				AnyTimes()

			// Prepare for unsubscribe from the second subscription
			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, dappID string, consumerIp string, analytics *metrics.RelayMetrics, chainMessage ChainMessage, directiveHeaders map[string]string, relayRequestData *pairingtypes.RelayPrivateData) (relayResult *common.RelayResult, errRet error) {
					wg.Done()
					return relayResult2, nil
				}).
				Times(2) // Should call SendParsedRelay, because it unsubscribed

			ctx = utils.WithUniqueIdentifier(ts.Ctx, utils.GenerateUniqueIdentifier())
			err = manager.UnsubscribeAll(ctx, dapp1, ts.Consumer.Addr.String(), nil)
			require.NoError(t, err)
			// Because the SendParsedRelay is called in a goroutine, we need to wait for it to finish
			wg.Wait()
		})
	}
}

func CreateConsumerSessionManager(chainID, apiInterface, consumerPublicAddress string) *lavasession.ConsumerSessionManager {
	rand.InitRandomSeed()
	baseLatency := common.AverageWorldLatency / 2 // we want performance to be half our timeout or better
	return lavasession.NewConsumerSessionManager(
		&lavasession.RPCEndpoint{NetworkAddress: "stub", ChainID: chainID, ApiInterface: apiInterface, TLSEnabled: false, HealthCheckPath: "/", Geolocation: 0},
		provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_BALANCED, 0, baseLatency, 1),
		nil, nil, consumerPublicAddress,
		lavasession.NewActiveSubscriptionProvidersStorage(),
	)
}
