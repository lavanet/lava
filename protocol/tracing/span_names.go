package tracing

// Chainlib listener root spans (extract trace context from external clients).
const (
	SpanChainlibHTTP = "chainlib.HandleHTTP"
	SpanChainlibGRPC = "chainlib.HandleGRPC"
)

// Consumer pipeline spans.
const (
	SpanConsumerSendRelay        = "consumer.SendRelay"
	SpanConsumerParseRelay       = "consumer.ParseRelay"
	SpanConsumerSendParsedRelay  = "consumer.SendParsedRelay"
	SpanConsumerProcessRelaySend = "consumer.ProcessRelaySend"
	SpanConsumerCacheLookup      = "consumer.CacheLookup"
	SpanConsumerGetSessions      = "consumer.GetSessions"
	SpanConsumerRelayInner       = "consumer.relayInner"
	SpanConsumerProcessingResult = "consumer.ProcessingResult"
)

// Provider pipeline spans.
const (
	SpanProviderHandleRelay       = "provider.HandleRelay"
	SpanProviderInitRelay         = "provider.InitRelay"
	SpanProviderValidateRequest   = "provider.ValidateRequest"
	SpanProviderCacheLookup       = "provider.CacheLookup"
	SpanProviderHandleConsistency = "provider.HandleConsistency"
	SpanProviderTryRelay          = "provider.TryRelay"
	SpanProviderSendUpstream      = "provider.SendUpstream"
	SpanProviderCacheStore        = "provider.CacheStore"
	SpanProviderFinalizeSession   = "provider.FinalizeSession"
)

// Chainproxy outbound spans — child of provider.SendUpstream when invoked
// from the provider relay path, or child of the caller's span otherwise.
// Each wraps an actual outbound HTTP/gRPC call to the upstream chain node.
const (
	SpanChainproxyJSONRPC       = "chainlib.sendJSONRPCRelay"
	SpanChainproxyREST          = "chainlib.sendRESTRelay"
	SpanChainproxyGRPC          = "chainlib.sendGRPCRelay"
	SpanChainproxyTendermintRPC = "chainlib.sendTendermintRPCRelay"
)

// Lava-chain communication spans.
const (
	SpanLavaChainQuery    = "lavachain.Query"
	SpanLavaChainSubmitTx = "lavachain.SubmitTx"
)

// Span attribute keys.
const (
	// Body recording attributes (gated by --otel-trace-body, set via
	// tracing.RecordBody).
	AttrRelayRequestBody  = "relay.request_body"
	AttrRelayResponseBody = "relay.response_body"
)

const (
	// Relay identification.
	attrRelayGUID         = "relay.guid"
	attrRelayChainID      = "relay.chain_id"
	attrRelayAPIInterface = "relay.api_interface"
	attrRelayMethod       = "relay.method"

	// attrRelayRetryCount is the total number of retry batches dispatched by
	// the state machine for a relay (failover + hedge). Cross-validation
	// parallel calls share a batch and do NOT increment this counter.
	attrRelayRetryCount = "relay.retry_count"

	// attrRelayAttempt is the 1-indexed batch number this provider call
	// belongs to. All parallel calls in the same batch (cross-validation)
	// share the same value.
	attrRelayAttempt = "relay.attempt"

	// Provider attributes.
	attrProviderAddress = "provider.address"

	// Cache attributes.
	attrCacheHit       = "cache.hit"
	attrCacheLatencyMs = "cache.latency_ms"

	// Session attributes.
	attrSessionRequested = "session.requested"
	attrSessionAcquired  = "session.acquired"

	// Consistency attributes.
	attrConsistencyTotal    = "consistency.total"
	attrConsistencyPassed   = "consistency.passed"
	attrConsistencyRejected = "consistency.rejected"
)

const (
	// New cache attribute (existing cache.* keys gain a role suffix to
	// disambiguate consumer vs provider cache spans on the same trace).
	attrCacheRole = "cache.role"

	// Provider HandleConsistency attributes.
	attrConsistencyTargetBlock = "consistency.target_block"
	attrConsistencyWaitedMs    = "consistency.waited_ms"
	attrConsistencyBailed      = "consistency.bailed"

	// Lava-chain query attributes.
	attrLavaChainMethod = "lavachain.method"
	attrLavaChainCode   = "lavachain.code"

	// Lava-chain tx attributes.
	attrLavaChainMsgType     = "lavachain.msg_type"
	attrLavaChainBroadcastMs = "lavachain.broadcast_ms"
	attrLavaChainCommitMs    = "lavachain.commit_ms"
	attrLavaChainHeight      = "lavachain.height"
	attrLavaChainTxHash      = "lavachain.tx_hash"
)
