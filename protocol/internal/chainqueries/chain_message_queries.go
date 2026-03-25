// Package chainqueries provides chain-message predicate helpers shared across
// protocol sub-packages. It is intentionally placed under protocol/internal so
// that packages outside the protocol tree cannot depend on it.
package chainqueries

import (
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
)

// IsArchiveRequest returns true if the chain message carries the archive extension.
func IsArchiveRequest(chainMessage chainlib.ChainMessage) bool {
	for _, ext := range chainMessage.GetExtensions() {
		if ext.Name == extensionslib.ArchiveExtension {
			return true
		}
	}
	return false
}

// IsDebugOrTraceRequest returns true if the request belongs to a debug or trace addon,
// as classified by the chain spec (AddOn = "debug" or "trace").
func IsDebugOrTraceRequest(chainMessage chainlib.ChainMessageForSend) bool {
	addon := chainlib.GetAddon(chainMessage)
	return addon == "debug" || addon == "trace"
}

// IsBatchRequest returns true if the chain message is a batch request (e.g., JSON-RPC batch).
func IsBatchRequest(chainMessage chainlib.ChainMessage) bool {
	return chainMessage.IsBatch()
}
