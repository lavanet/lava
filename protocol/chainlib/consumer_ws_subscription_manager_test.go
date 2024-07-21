package chainlib

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestConsumerWSSubscriptionManagerParallelSubscriptions(t *testing.T) {
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
	}

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ts := SetupForTests(t, 1, play.specId, "../../")

			dapp := "dapp"

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

			subscriptionIdExtractor := func(request ChainMessage, reply *rpcclient.JsonrpcMessage) string {
				return ""
			}

			// Create a new ConsumerWSSubscriptionManager
			manager := NewConsumerWSSubscriptionManager(consumerSessionManager, relaySender, nil, play.connectionType, chainParser, lavasession.NewActiveSubscriptionProvidersStorage(), subscriptionIdExtractor)

			wg := sync.WaitGroup{}
			wg.Add(10)
			// Start a new subscription for the first time, called SendParsedRelay once while in parallel calling 10 times subscribe with the same message
			// expected result is to have SendParsedRelay only once and 9 other messages waiting the broadcast.
			for i := 0; i < 10; i++ {
				// sending
				go func(index int) {
					ctx := utils.WithUniqueIdentifier(ts.Ctx, utils.GenerateUniqueIdentifier())
					var repliesChan <-chan *pairingtypes.RelayReply
					var firstReply *pairingtypes.RelayReply
					firstReply, repliesChan, err = manager.StartSubscription(ctx, chainMessage1, nil, nil, dapp+strconv.Itoa(index), ts.Consumer.Addr.String(), nil)
					go func() {
						for subMsg := range repliesChan {
							require.Equal(t, string(play.subscriptionFirstReply1), string(subMsg.Data))
						}
					}()
					assert.NoError(t, err)
					assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
					assert.NotNil(t, repliesChan)
					wg.Done()
				}(i)
			}
			wg.Wait()
		})
	}
}

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
		subscriptionId1          string
		subscriptionFirstReply1  []byte
		unsubscribeMessage1      []byte
		subscriptionRequestData2 []byte
		subscriptionId2          string
		subscriptionFirstReply2  []byte
		unsubscribeMessage2      []byte
	}{
		{
			name:           "Lava_TendermintRPC",
			specId:         "LAV1",
			apiInterface:   spectypes.APIInterfaceTendermintRPC,
			connectionType: "",

			subscriptionRequestData1: []byte(`{"jsonrpc":"2.0","id":3,"method":"subscribe","params":{"query":"tm.event='NewBlock'"}}`),
			subscriptionId1:          `{"query":"tm.event='NewBlock'"}`,
			subscriptionFirstReply1:  []byte(`{"jsonrpc":"2.0","id":3,"result":{}}`),
			unsubscribeMessage1:      []byte(`{"jsonrpc":"2.0","method":"unsubscribe","params":{"query":"tm.event='NewBlock'"},"id":1}`),

			subscriptionRequestData2: []byte(`{"jsonrpc":"2.0","id":4,"method":"subscribe","params":{"query":"tm.event= 'NewBlock'"}}`),
			subscriptionId2:          `{"query":"tm.event= 'NewBlock'"}`,
			subscriptionFirstReply2:  []byte(`{"jsonrpc":"2.0","id":4,"result":{}}`),
			unsubscribeMessage2:      []byte(`{"jsonrpc":"2.0","method":"unsubscribe","params":{"query":"tm.event= 'NewBlock'"},"id":1}`),
		},
		{
			name:           "Ethereum_JsonRPC",
			specId:         "ETH1",
			apiInterface:   spectypes.APIInterfaceJsonRPC,
			connectionType: "POST",

			subscriptionRequestData1: []byte(`{"jsonrpc":"2.0","id":5,"method":"eth_subscribe","params":["newHeads"]}`),
			subscriptionId1:          "0x1234567890",
			subscriptionFirstReply1:  []byte(`{"jsonrpc":"2.0","id":5,"result":["0x1234567890"]}`),
			unsubscribeMessage1:      []byte(`{"jsonrpc":"2.0","method":"eth_unsubscribe","params":["0x1234567890"],"id":1}`),

			subscriptionRequestData2: []byte(`{"jsonrpc":"2.0","id":6,"method":"eth_subscribe","params":["logs"]}`),
			subscriptionId2:          "0x2134567890",
			subscriptionFirstReply2:  []byte(`{"jsonrpc":"2.0","id":6,"result":["0x2134567890"]}`),
			unsubscribeMessage2:      []byte(`{"jsonrpc":"2.0","method":"eth_unsubscribe","params":["0x2134567890"],"id":1}`),
		},
		{
			name:           "StarkNet_JsonRPC",
			specId:         "STRK",
			apiInterface:   spectypes.APIInterfaceJsonRPC,
			connectionType: "POST",

			subscriptionRequestData1: []byte(`{"jsonrpc":"2.0","id":5,"method":"pathfinder_subscribe","params":["newHeads"]}`),
			subscriptionId1:          "0",
			subscriptionFirstReply1:  []byte(`{"jsonrpc":"2.0","id":5,"result":0}`),
			unsubscribeMessage1:      []byte(`{"jsonrpc":"2.0","method":"pathfinder_unsubscribe","params":[0],"id":1}`),

			subscriptionRequestData2: []byte(`{"jsonrpc":"2.0","id":6,"method":"pathfinder_subscribe","params":["logs"]}`),
			subscriptionId2:          "1",
			subscriptionFirstReply2:  []byte(`{"jsonrpc":"2.0","id":6,"result":1}`),
			unsubscribeMessage2:      []byte(`{"jsonrpc":"2.0","method":"pathfinder_unsubscribe","params":[1],"id":1}`),
		},
	}

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ts := SetupForTests(t, 1, play.specId, "../../")
			unsubscribeMessageWg := sync.WaitGroup{}
			ctx, cancel := context.WithCancel(ts.Ctx)
			defer func() {
				cancel()
				unsubscribeMessageWg.Wait()
			}()

			listenForExpectedMessages := func(ctx context.Context, repliesChan <-chan *pairingtypes.RelayReply, expectedMsg string) {
				select {
				case <-time.After(5 * time.Second):
					require.Fail(t, "Timeout waiting for messages", "Expected message: %s", expectedMsg)
					return
				case subMsg := <-repliesChan:
					require.Equal(t, expectedMsg, string(subMsg.Data))
				case <-ctx.Done():
					return
				}
			}

			expectNoMoreMessages := func(ctx context.Context, repliesChan <-chan *pairingtypes.RelayReply) {
				msgCounter := 0
				select {
				case <-ctx.Done():
					return
				case <-repliesChan:
					msgCounter++
					if msgCounter > 2 {
						require.Fail(t, "Unexpected message received")
					}
				}
			}

			dapp1 := "dapp1"
			dapp2 := "dapp2"

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			chainParser, _, _, _, _, err := CreateChainLibMocks(ts.Ctx, play.specId, play.apiInterface, nil, nil, "../../", nil)
			require.NoError(t, err)

			unsubscribeChainMessage1, err := chainParser.ParseMsg("", play.unsubscribeMessage1, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			subscribeChainMessage1, err := chainParser.ParseMsg("", play.subscriptionRequestData1, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			relaySender := NewMockRelaySender(ctrl)
			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
					relayPrivateData, ok := x.(*pairingtypes.RelayPrivateData)
					if !ok || relayPrivateData == nil {
						return false
					}

					if strings.Contains(string(relayPrivateData.Data), "unsubscribe") {
						unsubscribeMessageWg.Done()
					}

					// Always return false, because we don't to use this mock's default return for the calls
					return false
				})).
				AnyTimes()

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

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
					reqData, ok := x.(string)
					require.True(t, ok)
					areEqual := reqData == string(play.unsubscribeMessage1)
					return areEqual
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(unsubscribeChainMessage1, nil, &pairingtypes.RelayPrivateData{
					Data: play.unsubscribeMessage1,
				}, nil).
				AnyTimes()

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
					reqData, ok := x.(string)
					require.True(t, ok)
					areEqual := reqData == string(play.subscriptionRequestData1)
					return areEqual
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(subscribeChainMessage1, nil, nil, nil).
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

			expectedSubscriptionId := play.subscriptionId1
			subscriptionIdExtractor := func(request ChainMessage, reply *rpcclient.JsonrpcMessage) string {
				return expectedSubscriptionId
			}

			// Create a new ConsumerWSSubscriptionManager
			manager := NewConsumerWSSubscriptionManager(consumerSessionManager, relaySender, nil, play.connectionType, chainParser, lavasession.NewActiveSubscriptionProvidersStorage(), subscriptionIdExtractor)

			// Start a new subscription for the first time, called SendParsedRelay once
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan1, err := manager.StartSubscription(ctx, subscribeChainMessage1, nil, nil, dapp1, ts.Consumer.Addr.String(), nil)
			assert.NoError(t, err)
			unsubscribeMessageWg.Add(1)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.NotNil(t, repliesChan1)

			listenForExpectedMessages(ctx, repliesChan1, string(play.subscriptionFirstReply1))

			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(0) // Should not call SendParsedRelay, because it is already subscribed

			// Start a subscription again, same params, same dappKey, should not call SendParsedRelay
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan2, err := manager.StartSubscription(ctx, subscribeChainMessage1, nil, nil, dapp1, ts.Consumer.Addr.String(), nil)
			assert.NoError(t, err)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.Nil(t, repliesChan2) // Same subscription, same dappKey, no need for a new channel

			listenForExpectedMessages(ctx, repliesChan1, string(play.subscriptionFirstReply1))

			// Start a subscription again, same params, different dappKey, should not call SendParsedRelay
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan3, err := manager.StartSubscription(ctx, subscribeChainMessage1, nil, nil, dapp2, ts.Consumer.Addr.String(), nil)
			assert.NoError(t, err)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.NotNil(t, repliesChan3) // Same subscription, but different dappKey, so will create new channel

			listenForExpectedMessages(ctx, repliesChan1, string(play.subscriptionFirstReply1))
			listenForExpectedMessages(ctx, repliesChan3, string(play.subscriptionFirstReply1))

			// Prepare for the next subscription
			expectedSubscriptionId = play.subscriptionId2
			unsubscribeChainMessage2, err := chainParser.ParseMsg("", play.unsubscribeMessage2, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			subscribeChainMessage2, err := chainParser.ParseMsg("", play.subscriptionRequestData2, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
					reqData, ok := x.(string)
					require.True(t, ok)
					areEqual := reqData == string(play.unsubscribeMessage2)
					return areEqual
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(unsubscribeChainMessage2, nil, &pairingtypes.RelayPrivateData{
					Data: play.unsubscribeMessage2,
				}, nil).
				AnyTimes()

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
					reqData, ok := x.(string)
					require.True(t, ok)
					areEqual := reqData == string(play.subscriptionRequestData2)
					return areEqual
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(subscribeChainMessage2, nil, nil, nil).
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
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan4, err := manager.StartSubscription(ctx, subscribeChainMessage2, nil, nil, dapp1, ts.Consumer.Addr.String(), nil)
			assert.NoError(t, err)
			unsubscribeMessageWg.Add(1)
			assert.Equal(t, string(play.subscriptionFirstReply2), string(firstReply.Data))
			assert.NotNil(t, repliesChan4) // New subscription, new channel

			listenForExpectedMessages(ctx, repliesChan1, string(play.subscriptionFirstReply1))
			listenForExpectedMessages(ctx, repliesChan3, string(play.subscriptionFirstReply1))
			listenForExpectedMessages(ctx, repliesChan4, string(play.subscriptionFirstReply2))

			// Prepare for unsubscribe from the first subscription
			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(0) // Should call SendParsedRelay, because it unsubscribed

			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			err = manager.Unsubscribe(ctx, unsubscribeChainMessage1, nil, relayResult1.Request.RelayData, dapp2, ts.Consumer.Addr.String(), nil)
			require.NoError(t, err)

			listenForExpectedMessages(ctx, repliesChan1, string(play.subscriptionFirstReply1))
			expectNoMoreMessages(ctx, repliesChan3)
			listenForExpectedMessages(ctx, repliesChan4, string(play.subscriptionFirstReply2))

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

			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			err = manager.UnsubscribeAll(ctx, dapp1, ts.Consumer.Addr.String(), nil)
			require.NoError(t, err)

			expectNoMoreMessages(ctx, repliesChan1)
			expectNoMoreMessages(ctx, repliesChan3)
			expectNoMoreMessages(ctx, repliesChan4)

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
