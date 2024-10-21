package chainlib

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

const testGuid = "testGuid"

type RelayFinalizationBlocksHandlerMock struct{}

func (rf *RelayFinalizationBlocksHandlerMock) GetParametersForRelayDataReliability(
	ctx context.Context,
	request *pairingtypes.RelayRequest,
	chainMsg ChainMessage,
	relayTimeout time.Duration,
	blockLagForQosSync int64,
	averageBlockTime time.Duration,
	blockDistanceToFinalization,
	blocksInFinalizationData uint32,
) (latestBlock int64, requestedBlockHash []byte, requestedHashes []*chaintracker.BlockStore, modifiedReqBlock int64, finalized, updatedChainMessage bool, err error) {
	return 0, []byte{}, []*chaintracker.BlockStore{}, 0, true, true, nil
}

func (rf *RelayFinalizationBlocksHandlerMock) BuildRelayFinalizedBlockHashes(
	ctx context.Context,
	request *pairingtypes.RelayRequest,
	reply *pairingtypes.RelayReply,
	latestBlock int64,
	requestedHashes []*chaintracker.BlockStore,
	updatedChainMessage bool,
	relayTimeout time.Duration,
	averageBlockTime time.Duration,
	blockDistanceToFinalization uint32,
	blocksInFinalizationData uint32,
	modifiedReqBlock int64,
) (err error) {
	return nil
}

func TestSubscriptionManager_HappyFlow(t *testing.T) {
	playbook := []struct {
		name                    string
		specId                  string
		apiInterface            string
		connectionType          string
		subscriptionRequestData []byte
		subscriptionFirstReply  []byte
	}{
		{
			name:                    "TendermintRPC",
			specId:                  "LAV1",
			apiInterface:            spectypes.APIInterfaceTendermintRPC,
			connectionType:          "",
			subscriptionRequestData: []byte(`{"jsonrpc":"2.0","id":3,"method":"subscribe","params":{"query":"tm.event='NewBlock'"}}`),
			subscriptionFirstReply:  []byte(`{"jsonrpc":"2.0","id":3,"result":{}}`),
		},
		{
			name:                    "JsonRPC",
			specId:                  "ETH1",
			apiInterface:            spectypes.APIInterfaceJsonRPC,
			connectionType:          "POST",
			subscriptionRequestData: []byte(`{"jsonrpc":"2.0","id":5,"method":"eth_subscribe","params":["newHeads"]}`),
			subscriptionFirstReply:  []byte(`{"jsonrpc":"2.0","id":5,"result":"0x1234567890"}`),
		},
	}

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ts := SetupForTests(t, 1, play.specId, "../../")

			wg := sync.WaitGroup{}
			wg.Add(1)
			// msgCount := 0
			upgrader := websocket.Upgrader{}

			// Create a simple websocket server that mocks the node
			handleWebSocket := func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					require.NoError(t, err)
					return
				}
				defer conn.Close()

				for {
					// Read the request
					messageType, message, err := conn.ReadMessage()
					if err != nil {
						require.NoError(t, err)
						return
					}

					wg.Done()

					require.Equal(t, string(play.subscriptionRequestData)+"\n", string(message))

					// Write the first reply
					err = conn.WriteMessage(messageType, play.subscriptionFirstReply)
					if err != nil {
						require.NoError(t, err)
						return
					}
				}
			}

			chainParser, chainRouter, _, closeServer, _, err := CreateChainLibMocks(context.Background(), play.specId, play.apiInterface, nil, handleWebSocket, "../../", nil)
			require.NoError(t, err)
			if closeServer != nil {
				defer closeServer()
			}

			// Create the relay request and chain message
			relayRequest := &pairingtypes.RelayRequest{
				RelayData: &pairingtypes.RelayPrivateData{
					Data: play.subscriptionRequestData,
				},
				RelaySession: &pairingtypes.RelaySession{},
			}

			chainMessage, err := chainParser.ParseMsg("", play.subscriptionRequestData, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			// Create the provider node subscription manager
			mockRpcProvider := &RelayFinalizationBlocksHandlerMock{}
			pnsm := NewProviderNodeSubscriptionManager(chainRouter, chainParser, mockRpcProvider, ts.Providers[0].SK)

			consumerChannel := make(chan *pairingtypes.RelayReply)

			// Read the consumer channel that simulates consumer
			go func() {
				reply := <-consumerChannel
				require.NotNil(t, reply)
				require.Equal(t, string(play.subscriptionFirstReply), string(reply.Data))
			}()

			// Subscribe to the chain
			subscriptionId, err := pnsm.AddConsumer(ts.Ctx, relayRequest, chainMessage, ts.Consumer.Addr, consumerChannel, testGuid)
			require.NoError(t, err)
			require.NotEmpty(t, subscriptionId)

			wg.Wait() // Make sure the subscription manager sent a message to the node

			// Subscribe to the same subscription again, should return the same subscription id
			subscriptionIdNew, err := pnsm.AddConsumer(ts.Ctx, relayRequest, chainMessage, ts.Consumer.Addr, consumerChannel, testGuid)
			require.NoError(t, err)
			require.NotEmpty(t, subscriptionId)
			require.Equal(t, subscriptionId, subscriptionIdNew)

			// Cut the subscription, and re-subscribe, should send another message to node
			err = pnsm.RemoveConsumer(ts.Ctx, chainMessage, ts.Consumer.Addr, true, testGuid)
			require.NoError(t, err)

			// Make sure both the consumer channels are closed
			_, ok := <-consumerChannel
			require.False(t, ok)

			consumerChannel = make(chan *pairingtypes.RelayReply)
			waitTestToEnd := make(chan bool)
			// Read the consumer channel that simulates consumer
			go func() {
				defer func() { waitTestToEnd <- true }()
				reply := <-consumerChannel
				require.NotNil(t, reply)
				require.Equal(t, string(play.subscriptionFirstReply), string(reply.Data))
			}()

			wg.Add(1) // Should send another message to the node

			subscriptionId, err = pnsm.AddConsumer(ts.Ctx, relayRequest, chainMessage, ts.Consumer.Addr, consumerChannel, "testGuid")
			require.NoError(t, err)
			require.NotEmpty(t, subscriptionId)

			wg.Wait() // Make sure the subscription manager sent another message to the node

			// making sure our routine ended, otherwise the routine can read the wrong play.subscriptionFirstReply
			<-waitTestToEnd
		})
	}
}

func TestSubscriptionManager_MultipleParallelSubscriptionsWithTheSameParams(t *testing.T) {
	playbook := []struct {
		name                    string
		specId                  string
		apiInterface            string
		connectionType          string
		subscriptionRequestData []byte
		subscriptionFirstReply  []byte
	}{
		{
			name:                    "TendermintRPC",
			specId:                  "LAV1",
			apiInterface:            spectypes.APIInterfaceTendermintRPC,
			connectionType:          "",
			subscriptionRequestData: []byte(`{"jsonrpc":"2.0","id":3,"method":"subscribe","params":{"query":"tm.event='NewBlock'"}}`),
			subscriptionFirstReply:  []byte(`{"jsonrpc":"2.0","id":3,"result":{}}`),
		},
		{
			name:                    "JsonRPC",
			specId:                  "ETH1",
			apiInterface:            spectypes.APIInterfaceJsonRPC,
			connectionType:          "POST",
			subscriptionRequestData: []byte(`{"jsonrpc":"2.0","id":5,"method":"eth_subscribe","params":["newHeads"]}`),
			subscriptionFirstReply:  []byte(`{"jsonrpc":"2.0","id":5,"result":"0x1234567890"}`),
		},
	}

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ts := SetupForTests(t, 1, play.specId, "../../")

			wg := sync.WaitGroup{}
			// msgCount := 0
			upgrader := websocket.Upgrader{}

			// Create a simple websocket server that mocks the node
			handleWebSocket := func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					require.NoError(t, err)
					return
				}
				defer conn.Close()
				first := true
				for {
					// Read the request
					messageType, message, err := conn.ReadMessage()
					if err != nil {
						require.NoError(t, err)
						return
					}

					require.Equal(t, string(play.subscriptionRequestData)+"\n", string(message))

					if first { // on first reply we want some delay, so we can make sure the pending is working properly
						time.Sleep(time.Second * 2)
						first = false
					}
					utils.LavaFormatDebug("write message")
					wg.Done()

					// Write the first reply
					err = conn.WriteMessage(messageType, play.subscriptionFirstReply)
					if err != nil {
						require.NoError(t, err)
						return
					}
				}
			}

			chainParser, chainRouter, _, closeServer, _, err := CreateChainLibMocks(context.Background(), play.specId, play.apiInterface, nil, handleWebSocket, "../../", nil)
			require.NoError(t, err)
			if closeServer != nil {
				defer closeServer()
			}

			// Create the relay request and chain message
			relayRequest := &pairingtypes.RelayRequest{
				RelayData: &pairingtypes.RelayPrivateData{
					Data: play.subscriptionRequestData,
				},
				RelaySession: &pairingtypes.RelaySession{},
			}

			chainMessage, err := chainParser.ParseMsg("", play.subscriptionRequestData, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			// Create the provider node subscription manager
			mockRpcProvider := &RelayFinalizationBlocksHandlerMock{}
			pnsm := NewProviderNodeSubscriptionManager(chainRouter, chainParser, mockRpcProvider, ts.Providers[0].SK)

			wg.Add(1)
			wgAllIds := sync.WaitGroup{}
			wgAllIds.Add(10)
			for i := 0; i < 10; i++ {
				consumerChannel := make(chan *pairingtypes.RelayReply)
				// Read the consumer channel that simulates consumer
				go func() {
					reply := <-consumerChannel
					require.NotNil(t, reply)
					require.Equal(t, string(play.subscriptionFirstReply), string(reply.Data))
					wgAllIds.Done()
				}()
				// Subscribe to the chain
				go func(index int) {
					subscriptionId, err := pnsm.AddConsumer(ts.Ctx, relayRequest, chainMessage, ts.Consumer.Addr, consumerChannel, testGuid+strconv.Itoa(index))
					require.NoError(t, err)
					require.NotEmpty(t, subscriptionId)
				}(i)
			}

			utils.LavaFormatDebug("Waiting wait group")
			wgAllIds.Wait()
			wg.Wait() // Make sure the subscription manager sent a message to the node

			// Cut the subscription, and re-subscribe, should send another message to node
			err = pnsm.RemoveConsumer(ts.Ctx, chainMessage, ts.Consumer.Addr, true, testGuid)
			require.NoError(t, err)
		})
	}
}

func TestSubscriptionManager_MultipleParallelSubscriptionsWithTheSameParamsAndNodeMessageFailure(t *testing.T) {
	playbook := []struct {
		name                    string
		specId                  string
		apiInterface            string
		connectionType          string
		subscriptionRequestData []byte
		subscriptionFirstReply  []byte
	}{
		{
			name:                    "TendermintRPC",
			specId:                  "LAV1",
			apiInterface:            spectypes.APIInterfaceTendermintRPC,
			connectionType:          "",
			subscriptionRequestData: []byte(`{"jsonrpc":"2.0","id":3,"method":"subscribe","params":{"query":"tm.event='NewBlock'"}}`),
			subscriptionFirstReply:  []byte(`{"jsonrpc":"2.0","id":3,"result":{}}`),
		},
		{
			name:                    "JsonRPC",
			specId:                  "ETH1",
			apiInterface:            spectypes.APIInterfaceJsonRPC,
			connectionType:          "POST",
			subscriptionRequestData: []byte(`{"jsonrpc":"2.0","id":5,"method":"eth_subscribe","params":["newHeads"]}`),
			subscriptionFirstReply:  []byte(`{"jsonrpc":"2.0","id":5,"result":"0x1234567890"}`),
		},
	}

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			ts := SetupForTests(t, 1, play.specId, "../../")

			sentMessageToNodeChannel := make(chan bool, 1)
			// msgCount := 0
			upgrader := websocket.Upgrader{}
			first := true
			// Create a simple websocket server that mocks the node
			handleWebSocket := func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					require.NoError(t, err)
					return
				}
				defer conn.Close()
				for {
					// Read the request
					messageType, message, err := conn.ReadMessage()
					if err != nil {
						require.NoError(t, err)
						return
					}

					require.Equal(t, string(play.subscriptionRequestData)+"\n", string(message))

					if first { // on first reply we want some delay, so we can make sure the pending is working properly
						time.Sleep(time.Second * 2)
						first = false
						conn.Close() // first connection should fail.
						return
					}
					utils.LavaFormatDebug("write message")
					select {
					case sentMessageToNodeChannel <- true:
						utils.LavaFormatDebug("sent message to node")
					default:
						utils.LavaFormatDebug("unable to communicate with the test")
					}

					// Write the first reply
					err = conn.WriteMessage(messageType, play.subscriptionFirstReply)
					if err != nil {
						require.NoError(t, err)
						return
					}
				}
			}

			chainParser, chainRouter, _, closeServer, _, err := CreateChainLibMocks(context.Background(), play.specId, play.apiInterface, nil, handleWebSocket, "../../", nil)
			require.NoError(t, err)
			if closeServer != nil {
				defer closeServer()
			}

			// Create the relay request and chain message
			relayRequest := &pairingtypes.RelayRequest{
				RelayData: &pairingtypes.RelayPrivateData{
					Data: play.subscriptionRequestData,
				},
				RelaySession: &pairingtypes.RelaySession{},
			}

			chainMessage, err := chainParser.ParseMsg("", play.subscriptionRequestData, play.connectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
			require.NoError(t, err)

			// Create the provider node subscription manager
			mockRpcProvider := &RelayFinalizationBlocksHandlerMock{}
			pnsm := NewProviderNodeSubscriptionManager(chainRouter, chainParser, mockRpcProvider, ts.Providers[0].SK)

			wgAllIds := sync.WaitGroup{}
			wgAllIds.Add(9)
			errors := []error{}
			for i := 0; i < 10; i++ {
				consumerChannel := make(chan *pairingtypes.RelayReply)
				// Read the consumer channel that simulates consumer
				go func() {
					reply := <-consumerChannel
					require.NotNil(t, reply)
					require.Equal(t, string(play.subscriptionFirstReply), string(reply.Data))
					wgAllIds.Done()
				}()
				// Subscribe to the chain
				go func(index int) {
					_, err := pnsm.AddConsumer(ts.Ctx, relayRequest, chainMessage, ts.Consumer.Addr, consumerChannel, testGuid+strconv.Itoa(index))
					if err != nil {
						errors = append(errors, err)
					}
				}(i)
			}

			utils.LavaFormatDebug("Waiting wait group")
			wgAllIds.Wait()
			select {
			case <-sentMessageToNodeChannel: // Make sure the subscription manager sent a message to the node
			case <-time.After(time.Second * 10):
				require.Fail(t, "timeout waiting for message to node")
			}
			// make sure we had only one error, on the first subscription attempt
			require.Len(t, errors, 1)

			// Cut the subscription, and re-subscribe, should send another message to node
			err = pnsm.RemoveConsumer(ts.Ctx, chainMessage, ts.Consumer.Addr, true, testGuid)
			require.NoError(t, err)
		})
	}
}
