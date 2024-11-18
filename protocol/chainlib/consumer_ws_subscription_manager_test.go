package chainlib

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/protocol/provideroptimizer"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/rand"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomockuber "go.uber.org/mock/gomock"
)

const (
	numberOfParallelSubscriptions = 10
	uniqueId                      = "1234"
	projectHashTest               = "test_projecthash"
	chainIdTest                   = "test_chainId"
	apiTypeTest                   = "test_apiType"
)

func TestConsumerWSSubscriptionManagerParallelSubscriptionsOnSameDappIdIp(t *testing.T) {
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
	metricsData := metrics.NewRelayAnalytics(projectHashTest, chainIdTest, apiTypeTest)
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ts := SetupForTests(t, 1, play.specId, "../../")

			dapp := "dapp"
			ip := "127.0.0.1"

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			chainParser, _, _, _, _, err := CreateChainLibMocks(ts.Ctx, play.specId, play.apiInterface, nil, nil, "../../", nil)
			require.NoError(t, err)

			chainMessage1, err := chainParser.ParseMsg("", play.subscriptionRequestData1, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			protocolMessage1 := NewProtocolMessage(chainMessage1, nil, nil, dapp, ip)
			relaySender := NewMockRelaySender(ctrl)
			relaySender.
				EXPECT().
				CreateDappKey(gomock.Any()).
				DoAndReturn(func(userData common.UserData) string {
					return userData.DappId + userData.ConsumerIp
				}).
				AnyTimes()
			relaySender.
				EXPECT().
				SetConsistencySeenBlock(gomock.Any(), gomock.Any()).
				AnyTimes()

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(protocolMessage1, nil).
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
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(1) // Should call SendParsedRelay, because it is the first time we subscribe

			consumerSessionManager := CreateConsumerSessionManager(play.specId, play.apiInterface, ts.Consumer.Addr.String())

			// Create a new ConsumerWSSubscriptionManager
			manager := NewConsumerWSSubscriptionManager(consumerSessionManager, relaySender, nil, play.connectionType, chainParser, lavasession.NewActiveSubscriptionProvidersStorage(), nil)
			uniqueIdentifiers := make([]string, numberOfParallelSubscriptions)
			wg := sync.WaitGroup{}
			wg.Add(numberOfParallelSubscriptions)

			// Start a new subscription for the first time, called SendParsedRelay once while in parallel calling 10 times subscribe with the same message
			// expected result is to have SendParsedRelay only once and 9 other messages waiting the broadcast.
			for i := 0; i < numberOfParallelSubscriptions; i++ {
				uniqueIdentifiers[i] = strconv.FormatUint(utils.GenerateUniqueIdentifier(), 10)
				// sending
				go func(index int) {
					ctx := utils.WithUniqueIdentifier(ts.Ctx, utils.GenerateUniqueIdentifier())
					var repliesChan <-chan *pairingtypes.RelayReply
					var firstReply *pairingtypes.RelayReply

					firstReply, repliesChan, err = manager.StartSubscription(ctx, protocolMessage1, dapp, ip, uniqueIdentifiers[index], metricsData)
					go func() {
						for subMsg := range repliesChan {
							// utils.LavaFormatInfo("got reply for index", utils.LogAttr("index", index))
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

			// now we have numberOfParallelSubscriptions subscriptions currently running
			require.Len(t, manager.connectedDapps, numberOfParallelSubscriptions)
			// remove one
			err = manager.Unsubscribe(ts.Ctx, protocolMessage1, dapp, ip, uniqueIdentifiers[0], metricsData)
			require.NoError(t, err)
			// now we have numberOfParallelSubscriptions - 1
			require.Len(t, manager.connectedDapps, numberOfParallelSubscriptions-1)
			// check we still have an active subscription.
			require.Len(t, manager.activeSubscriptions, 1)

			// same flow for unsubscribe all
			err = manager.UnsubscribeAll(ts.Ctx, dapp, ip, uniqueIdentifiers[1], metricsData)
			require.NoError(t, err)
			// now we have numberOfParallelSubscriptions - 2
			require.Len(t, manager.connectedDapps, numberOfParallelSubscriptions-2)
			// check we still have an active subscription.
			require.Len(t, manager.activeSubscriptions, 1)
		})
	}
}

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
			protocolMessage1 := NewProtocolMessage(chainMessage1, nil, nil, dapp, "")
			relaySender := NewMockRelaySender(ctrl)
			relaySender.
				EXPECT().
				CreateDappKey(gomock.Any()).
				DoAndReturn(func(userData common.UserData) string {
					return userData.DappId + userData.ConsumerIp
				}).
				AnyTimes()
			relaySender.
				EXPECT().
				SetConsistencySeenBlock(gomock.Any(), gomock.Any()).
				AnyTimes()

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(protocolMessage1, nil).
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
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(1) // Should call SendParsedRelay, because it is the first time we subscribe

			consumerSessionManager := CreateConsumerSessionManager(play.specId, play.apiInterface, ts.Consumer.Addr.String())
			metricsData := metrics.NewRelayAnalytics(projectHashTest, chainIdTest, apiTypeTest)
			// Create a new ConsumerWSSubscriptionManager
			manager := NewConsumerWSSubscriptionManager(consumerSessionManager, relaySender, nil, play.connectionType, chainParser, lavasession.NewActiveSubscriptionProvidersStorage(), nil)

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
					firstReply, repliesChan, err = manager.StartSubscription(ctx, protocolMessage1, dapp+strconv.Itoa(index), ts.Consumer.Addr.String(), uniqueId, metricsData)
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

func TestRateLimit(t *testing.T) {
	numberOfRequests := &atomic.Uint64{}
	fmt.Println(numberOfRequests.Load())
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
	}
	metricsData := metrics.NewRelayAnalytics(projectHashTest, chainIdTest, apiTypeTest)

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

			subscribeProtocolMessage1 := NewProtocolMessage(subscribeChainMessage1, nil, nil, dapp1, ts.Consumer.Addr.String())
			unsubscribeProtocolMessage1 := NewProtocolMessage(unsubscribeChainMessage1, nil, &pairingtypes.RelayPrivateData{
				Data: play.unsubscribeMessage1,
			}, dapp1, ts.Consumer.Addr.String(),
			)
			relaySender := NewMockRelaySender(ctrl)
			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomockuber.Cond(func(x any) bool {
					protocolMsg, ok := x.(ProtocolMessage)
					require.True(t, ok)
					require.NotNil(t, protocolMsg)
					if protocolMsg.RelayPrivateData() == nil {
						return false
					}
					if strings.Contains(string(protocolMsg.RelayPrivateData().Data), "unsubscribe") {
						unsubscribeMessageWg.Done()
					}

					// Always return false, because we don't to use this mock's default return for the calls
					return false
				})).
				AnyTimes()

			relaySender.
				EXPECT().
				CreateDappKey(gomock.Any()).
				DoAndReturn(func(userData common.UserData) string {
					return userData.DappId + userData.ConsumerIp
				}).
				AnyTimes()

			relaySender.
				EXPECT().
				SetConsistencySeenBlock(gomock.Any(), gomock.Any()).
				AnyTimes()

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomockuber.Cond(func(x any) bool {
					reqData, ok := x.(string)
					require.True(t, ok)
					areEqual := reqData == string(play.unsubscribeMessage1)
					return areEqual
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(unsubscribeProtocolMessage1, nil).
				AnyTimes()

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomockuber.Cond(func(x any) bool {
					reqData, ok := x.(string)
					require.True(t, ok)
					areEqual := reqData == string(play.subscriptionRequestData1)
					return areEqual
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(subscribeProtocolMessage1, nil).
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
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(1) // Should call SendParsedRelay, because it is the first time we subscribe

			consumerSessionManager := CreateConsumerSessionManager(play.specId, play.apiInterface, ts.Consumer.Addr.String())

			// Create a new ConsumerWSSubscriptionManager
			manager := NewConsumerWSSubscriptionManager(consumerSessionManager, relaySender, nil, play.connectionType, chainParser, lavasession.NewActiveSubscriptionProvidersStorage(), nil)

			// Start a new subscription for the first time, called SendParsedRelay once
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())

			firstReply, repliesChan1, err := manager.StartSubscription(ctx, subscribeProtocolMessage1, dapp1, ts.Consumer.Addr.String(), uniqueId, metricsData)
			assert.NoError(t, err)
			unsubscribeMessageWg.Add(1)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.NotNil(t, repliesChan1)

			listenForExpectedMessages(ctx, repliesChan1, string(play.subscriptionFirstReply1))

			relaySender.
				EXPECT().
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(0) // Should not call SendParsedRelay, because it is already subscribed

			// Start a subscription again, same params, same dappKey, should not call SendParsedRelay
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan2, err := manager.StartSubscription(ctx, subscribeProtocolMessage1, dapp1, ts.Consumer.Addr.String(), uniqueId, metricsData)
			assert.NoError(t, err)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.Nil(t, repliesChan2) // Same subscription, same dappKey, no need for a new channel

			listenForExpectedMessages(ctx, repliesChan1, string(play.subscriptionFirstReply1))

			// Start a subscription again, same params, different dappKey, should not call SendParsedRelay
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			firstReply, repliesChan3, err := manager.StartSubscription(ctx, subscribeProtocolMessage1, dapp2, ts.Consumer.Addr.String(), uniqueId, metricsData)
			assert.NoError(t, err)
			assert.Equal(t, string(play.subscriptionFirstReply1), string(firstReply.Data))
			assert.NotNil(t, repliesChan3) // Same subscription, but different dappKey, so will create new channel

			listenForExpectedMessages(ctx, repliesChan1, string(play.subscriptionFirstReply1))
			listenForExpectedMessages(ctx, repliesChan3, string(play.subscriptionFirstReply1))

			// Prepare for the next subscription
			unsubscribeChainMessage2, err := chainParser.ParseMsg("", play.unsubscribeMessage2, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			unsubscribeProtocolMessage2 := NewProtocolMessage(unsubscribeChainMessage2, nil, &pairingtypes.RelayPrivateData{Data: play.unsubscribeMessage2}, dapp2, ts.Consumer.Addr.String())
			subscribeChainMessage2, err := chainParser.ParseMsg("", play.subscriptionRequestData2, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)
			subscribeProtocolMessage2 := NewProtocolMessage(subscribeChainMessage2, nil, nil, dapp2, ts.Consumer.Addr.String())
			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomockuber.Cond(func(x any) bool {
					reqData, ok := x.(string)
					require.True(t, ok)
					areEqual := reqData == string(play.unsubscribeMessage2)
					return areEqual
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(unsubscribeProtocolMessage2, nil).
				AnyTimes()

			relaySender.
				EXPECT().
				ParseRelay(gomock.Any(), gomock.Any(), gomockuber.Cond(func(x any) bool {
					reqData, ok := x.(string)
					require.True(t, ok)
					areEqual := reqData == string(play.subscriptionRequestData2)
					return areEqual
				}), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(subscribeProtocolMessage2, nil).
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
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult2, nil).
				Times(1) // Should call SendParsedRelay, because it is the first time we subscribe

			// Start a subscription again, different params, same dappKey, should call SendParsedRelay
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())

			firstReply, repliesChan4, err := manager.StartSubscription(ctx, subscribeProtocolMessage2, dapp1, ts.Consumer.Addr.String(), uniqueId, metricsData)
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
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(relayResult1, nil).
				Times(0) // Should call SendParsedRelay, because it unsubscribed

			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			unsubProtocolMessage := NewProtocolMessage(unsubscribeChainMessage1, nil, relayResult1.Request.RelayData, dapp2, ts.Consumer.Addr.String())
			err = manager.Unsubscribe(ctx, unsubProtocolMessage, dapp2, ts.Consumer.Addr.String(), uniqueId, metricsData)
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
				SendParsedRelay(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, analytics *metrics.RelayMetrics, protocolMessage ProtocolMessage) (relayResult *common.RelayResult, errRet error) {
					wg.Done()
					return relayResult2, nil
				}).
				Times(2) // Should call SendParsedRelay, because it unsubscribed

			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			err = manager.UnsubscribeAll(ctx, dapp1, ts.Consumer.Addr.String(), uniqueId, metricsData)
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
		provideroptimizer.NewProviderOptimizer(provideroptimizer.STRATEGY_BALANCED, 0, baseLatency, 1, nil, "dontcare"),
		nil, nil, consumerPublicAddress,
		lavasession.NewActiveSubscriptionProvidersStorage(),
	)
}
