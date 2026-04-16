package common

// SolanaBlockNotAvailableCode is the JSON-RPC error code for
// "Block not available for slot X", indicating a skipped slot or propagation delay.
const SolanaBlockNotAvailableCode = -32004

// IsSolanaFamily returns true if the chain ID belongs to the Solana/SVM family.
// These chains use slot-based block numbering where slots can be skipped or
// not yet propagated, requiring special handling in block fetching.
//
// Delegates to the authoritative chainFamilyMap in error_registry.go so adding
// a new SVM chain only needs to update that single map — no parallel switch
// to keep in sync.
func IsSolanaFamily(chainID string) bool {
	family, ok := GetChainFamily(chainID)
	return ok && family == ChainFamilySolana
}
