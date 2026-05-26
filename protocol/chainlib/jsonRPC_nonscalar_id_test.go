package chainlib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

// TestSendNodeMsg_NonScalarRequestId is an end-to-end regression test for the JSON-RPC
// id-validation fix. It drives the real ParseMsg -> SendNodeMsg path against a mock node.
//
// A client may send a non-scalar (object/array) request id — invalid per JSON-RPC 2.0,
// e.g. a .NET CancellationToken serialized into the id. Two node behaviours were observed
// against live nodes, and both must be handled without failing the relay:
//
//   - NEAR-like: the node accepts the id, returns a valid result, and echoes the object id
//     back. The valid result must be returned, not discarded.
//   - Polygon-like: the node rejects the id per spec and replies id:null + "invalid request".
//     That clean node error must pass through to the consumer (no spurious "ID mismatch" or
//     cross-provider retry storm).
//
// Before the fix both cases hard-failed with "failed parsing ID ... not a string or float".
func TestSendNodeMsg_NonScalarRequestId(t *testing.T) {
	ctx := context.Background()
	// The client sends a non-scalar (object) id — the malformed-client case.
	objIDRequest := []byte(`{"jsonrpc":"2.0","id":{"x":1},"method":"eth_blockNumber","params":[]}`)
	wsNoop := func(string) string { return `{"jsonrpc":"2.0","id":1,"result":"0x10a7a08"}` }

	requestID := func(body []byte) string {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.Unmarshal(body, &req)
		return strings.TrimSpace(string(req.ID))
	}

	t.Run("node echoes object id and returns a result (NEAR-like)", func(t *testing.T) {
		// GIVEN a node that echoes the request id verbatim and returns a valid result
		node := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			id := requestID(body)
			if id == "" {
				id = "null"
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x10a7a08"}`, id)
		})
		chainParser, chainRouter, _, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, node, createWebSocketHandler(wsNoop), "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)

		// WHEN a request carrying a non-scalar id is relayed
		chainMessage, err := chainParser.ParseMsg("", objIDRequest, "POST", nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		reply, _, _, _, _, err := chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)

		// THEN the relay succeeds and the valid result is returned (not discarded on an id parse error)
		require.NoError(t, err)
		require.NotNil(t, reply)
		require.Contains(t, string(reply.RelayReply.Data), "0x10a7a08")
	})

	t.Run("node rejects object id with id:null (Polygon-like)", func(t *testing.T) {
		// GIVEN a node that rejects a non-scalar id per spec (id:null + invalid request)
		node := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			if id := requestID(body); strings.HasPrefix(id, "{") || strings.HasPrefix(id, "[") {
				fmt.Fprint(w, `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"invalid request"}}`)
			} else {
				fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x10a7a08"}`)
			}
		})
		chainParser, chainRouter, _, closeServer, _, err := CreateChainLibMocks(ctx, "ETH1", spectypes.APIInterfaceJsonRPC, node, createWebSocketHandler(wsNoop), "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)

		// WHEN a request carrying a non-scalar id is relayed
		chainMessage, err := chainParser.ParseMsg("", objIDRequest, "POST", nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		reply, _, _, _, _, err := chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)

		// THEN the relay does not hard-fail; the node's own error passes through to the consumer
		require.NoError(t, err)
		require.NotNil(t, reply)
		require.Contains(t, string(reply.RelayReply.Data), "invalid request")
	})
}
