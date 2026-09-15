package rpcclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestIsStarknetNotification(t *testing.T) {
	notification := &JsonrpcMessage{
		Version: Vsn,
		Method:  "starknet_subscriptionNewHeads",
		Params:  json.RawMessage(`{"subscription_id":"340282366920938463463374607431768211456","result":{"block_number":5}}`),
	}
	require.True(t, notification.isStarknetNotification())
	require.False(t, notification.isEthereumNotification())
	require.False(t, notification.isStarkNetPathfinderNotification())
	require.False(t, notification.isTendermintNotification())

	for _, method := range []string{
		"starknet_subscriptionEvents",
		"starknet_subscriptionTransactionStatus",
		"starknet_subscriptionNewTransactionReceipts",
		"starknet_subscriptionNewTransaction",
		"starknet_subscriptionReorg",
	} {
		msg := &JsonrpcMessage{Version: Vsn, Method: method, Params: json.RawMessage(`{"subscription_id":"1","result":{}}`)}
		require.True(t, msg.isStarknetNotification(), method)
	}

	// ethereum-style notification must not match
	ethNotification := &JsonrpcMessage{
		Version: Vsn,
		Method:  "eth_subscription",
		Params:  json.RawMessage(`{"subscription":"0x1","result":{}}`),
	}
	require.False(t, ethNotification.isStarknetNotification())

	// a starknet_subscribe* response has no method, must not match
	response := &JsonrpcMessage{
		Version: Vsn,
		ID:      json.RawMessage(`1`),
		Result:  json.RawMessage(`"340282366920938463463374607431768211456"`),
	}
	require.False(t, response.isStarknetNotification())

	// legacy pathfinder notification (top-level result) must not match
	pathfinderNotification := &JsonrpcMessage{
		Version: Vsn,
		Method:  "pathfinder_subscription",
		Result:  json.RawMessage(`{"subscription":1,"result":{}}`),
	}
	require.False(t, pathfinderNotification.isStarknetNotification())
}

// TestStarknetSubscriptionEndToEnd subscribes against a mock starknet node over
// websocket using by-name params (the canonical starknet form) and verifies the
// v0.9+ notification envelope {method: starknet_subscriptionXxx, params:
// {subscription_id, result}} is delivered to the subscription channel.
func TestStarknetSubscriptionEndToEnd(t *testing.T) {
	subId := "340282366920938463463374607431768211456"
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, reqData, err := conn.ReadMessage()
		require.NoError(t, err)
		var req JsonrpcMessage
		require.NoError(t, json.Unmarshal(reqData, &req))
		require.Equal(t, "starknet_subscribeNewHeads", req.Method)
		require.JSONEq(t, `{"block_id":"latest"}`, string(req.Params))

		resp, err := json.Marshal(JsonrpcMessage{Version: Vsn, ID: req.ID, Result: json.RawMessage(`"` + subId + `"`)})
		require.NoError(t, err)
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, resp))

		// a notification for an unknown subscription id must be dropped silently
		unknown := `{"jsonrpc":"2.0","method":"starknet_subscriptionNewHeads","params":{"subscription_id":"999","result":{"block_number":4}}}`
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(unknown)))

		notification := `{"jsonrpc":"2.0","method":"starknet_subscriptionNewHeads","params":{"subscription_id":"` + subId + `","result":{"block_number":5}}}`
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(notification)))

		// keep the connection open until the client disconnects
		conn.ReadMessage()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := DialWebsocket(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	require.NoError(t, err)
	defer client.Close()

	ch := make(chan *JsonrpcMessage, 8)
	sub, resp, err := client.Subscribe(ctx, nil, "starknet_subscribeNewHeads", ch, map[string]interface{}{"block_id": "latest"})
	require.NoError(t, err)
	require.NotNil(t, sub)
	require.NotNil(t, resp)
	require.Equal(t, `"`+subId+`"`, string(resp.Result))
	defer sub.Unsubscribe()

	select {
	case msg := <-ch:
		require.Equal(t, "starknet_subscriptionNewHeads", msg.Method)
		var result starknetSubscriptionResult
		require.NoError(t, json.Unmarshal(msg.Params, &result))
		require.Equal(t, subId, result.ID)
		require.JSONEq(t, `{"block_number":5}`, string(result.Result))
	case <-time.After(3 * time.Second):
		t.Fatal("starknet notification was not delivered to the subscription channel")
	}

	// the unknown-id notification must not have been delivered
	select {
	case msg := <-ch:
		t.Fatalf("unexpected extra message delivered: %+v", msg)
	case <-time.After(200 * time.Millisecond):
	}
}
