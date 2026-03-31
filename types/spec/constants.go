package spec

const (
	NOT_APPLICABLE  int64 = -1
	LATEST_BLOCK    int64 = -2
	EARLIEST_BLOCK  int64 = -3
	PENDING_BLOCK   int64 = -4
	SAFE_BLOCK      int64 = -5
	FINALIZED_BLOCK int64 = -6

	DEFAULT_PARSED_RESULT_INDEX = 0

	APIInterfaceJsonRPC       = "jsonrpc"
	APIInterfaceTendermintRPC = "tendermintrpc"
	APIInterfaceRest          = "rest"
	APIInterfaceGrpc          = "grpc"

	ParserArgLatest = "latest"

	EncodingBase64 = "base64"
	EncodingHex    = "hex"

	// Event names
	ParamChangeEventName = "param_change"
	SpecAddEventName     = "spec_add"
	SpecModifyEventName  = "spec_modify"
	SpecRefreshEventName = "spec_refresh"
)

// IsFinalizedBlock returns true when the requested block is old enough to be
// considered finalized relative to the latest known block and the chain's
// finalization criteria (number of confirmations required).
func IsFinalizedBlock(requestedBlock, latestBlock, finalizationCriteria int64) bool {
	switch requestedBlock {
	case NOT_APPLICABLE:
		return false
	default:
		if requestedBlock < 0 {
			return false
		}
		if requestedBlock <= latestBlock-finalizationCriteria {
			return true
		}
	}
	return false
}
