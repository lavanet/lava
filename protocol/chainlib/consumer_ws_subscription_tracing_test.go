package chainlib

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWSPushEventForwardingHasNoTracing is a regression guard for the OTel
// tracing design: server-pushed subscription events (the long-lived stream
// from provider to consumer to WS client) must NOT create per-message spans.
// Each pushed event in a high-volume subscription (e.g. eth_subscribe newHeads
// on a busy chain) would otherwise produce ~10-15 spans, flooding the trace
// backend.
//
// The test parses consumer_ws_subscription_manager.go and asserts that the
// push-event-forwarding functions contain no calls to tracing.Start* helpers.
// If a future change accidentally adds tracing on the push path, this test
// fails before the change ships.
//
// Functions audited:
//   - listenForSubscriptionMessages: contains the replyServer.RecvMsg loop
//     that reads pushed events from the provider's gRPC stream.
//   - handleIncomingSubscriptionNodeMessage: called for each received reply
//     to forward it to the WS client.
//   - verifySubscriptionMessage: signature verification on each pushed reply.
//
// Client-initiated message paths (e.g. StartSubscription, Unsubscribe,
// sendUnsubscribeMessage) are NOT audited — those represent real client
// operations that SHOULD be traced via ParseRelay/SendParsedRelay, matching
// the spec's "trace handshake-class events" intent.
func TestWSPushEventForwardingHasNoTracing(t *testing.T) {
	const file = "consumer_ws_subscription_manager.go"
	pushEventFunctions := map[string]bool{
		"listenForSubscriptionMessages":        true,
		"handleIncomingSubscriptionNodeMessage": true,
		"verifySubscriptionMessage":            true,
	}

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	require.NoError(t, err, "could not parse %s — has the file moved?", file)

	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil {
			continue
		}
		if !pushEventFunctions[fn.Name.Name] {
			continue
		}

		// Walk the function body looking for tracing.Start* call expressions.
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if pkg.Name == "tracing" && strings.HasPrefix(sel.Sel.Name, "Start") {
				pos := fset.Position(call.Pos())
				t.Errorf(
					"%s:%d — found tracing.%s call inside %s; push-event forwarding must not produce per-message spans (see spec section 3.5)",
					pos.Filename, pos.Line, sel.Sel.Name, fn.Name.Name,
				)
			}
			return true
		})

		delete(pushEventFunctions, fn.Name.Name)
	}

	// Catch renames: every audited function must have been found.
	for fnName := range pushEventFunctions {
		t.Errorf("audited function %q not found in %s — has it been renamed? Update the test or the rename is a real bug.", fnName, file)
	}
}
