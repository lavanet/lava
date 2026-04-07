package tracing

// Span names — each matches the function or operation it instruments.
const (
	SpanSendRelay                    = "smartrouter.SendRelay"
	SpanParsedRelay                  = "smartrouter.SendParsedRelay"
	SpanParseRelay                   = "smartrouter.ParseRelay"
	SpanProcessRelaySend             = "smartrouter.ProcessRelaySend"
	SpanProcessingResult             = "smartrouter.ProcessingResult"
	SpanCacheLookup                  = "smartrouter.CacheLookup"
	SpanGetSessions                  = "smartrouter.GetSessions"
	SpanFilterEndpointsByConsistency = "smartrouter.filterEndpointsByConsistency"
	SpanRelayInnerDirect             = "smartrouter.relayInnerDirect"
	SpanSendJSONRPCRelay             = "smartrouter.sendJSONRPCRelay"
	SpanSendRESTRelay                = "smartrouter.sendRESTRelay"
	SpanSendGRPCRelay                = "smartrouter.sendGRPCRelay"
)

// Span attribute keys.
const (
	// Body recording attributes (gated by --otel-trace-body, set from
	// rpcsmartrouter via tracing.RecordBody).
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
	// the state machine for a relay (failover + hedge). Set on
	// smartrouter.ProcessRelaySend. Cross-validation parallel calls share a
	// batch and do NOT increment this counter.
	attrRelayRetryCount = "relay.retry_count"

	// attrRelayAttempt is the 1-indexed batch number this provider call
	// belongs to. All parallel calls in the same batch (cross-validation)
	// share the same value. Set on smartrouter.relayInnerDirect.
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
