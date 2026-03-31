package common

// IsSolanaFamily returns true if the chain ID belongs to the Solana/SVM family.
// These chains use slot-based block numbering where slots can be skipped or
// not yet propagated, requiring special handling in block fetching.
func IsSolanaFamily(chainID string) bool {
	switch chainID {
	case "SOLANA", "SOLANAT", "KOII", "KOIIT":
		return true
	default:
		return false
	}
}
