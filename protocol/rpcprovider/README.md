# RPC Provider

The RPC Provider serves blockchain RPC requests from consumers on the Lava network.

## Observability / Tracing

`rpcprovider` emits OpenTelemetry traces by default when an OTLP endpoint is
configured. Disable with `OTEL_SDK_DISABLED=true` or `OTEL_TRACES_EXPORTER=none`.

Spans cover the relay pipeline (handle, validate, cache lookup, consistency
wait, upstream call, finalize) plus background lava-chain communication
(`lavachain.Query` for spec/pairing fetches, `lavachain.SubmitTx` for
relay-payment claims). Trace context arrives via the consumer→provider gRPC
hop and is propagated into the upstream chain-node call as a `traceparent`
HTTP header (or gRPC metadata).

### Quick start

```bash
OTEL_SERVICE_NAME=my-provider-eu-1 \
OTEL_TRACES_EXPORTER=otlp \
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel.example.com:4317 \
./lavap rpcprovider ...
```

If `OTEL_SERVICE_NAME` is unset, the binary uses `lava-provider` as its in-code
default, which keeps it distinct from `lava-consumer` even when both run on
the same host.

### Recording request/response bodies

Add `--otel-trace-body` to record relay request and response bodies on the
provider's relay spans. Body size is bounded by
`OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT` (default: unlimited; set to e.g.
`4096` for 4 KiB cap).

**Caution for production providers:** relay payloads can be large (e.g.
`eth_getLogs` responses in megabytes). Enabling body recording on a high-volume
provider materially increases trace-export bandwidth and collector ingestion
cost. Pair with a non-unlimited length limit or restrict to debug sessions.

### Cache backend

If running with `--cache-be <addr>`, the cache-lookup span carries
`cache.hit=true|false` and a millisecond latency attribute. With no cache
backend running, the provider correctly bypasses caching entirely — no
`provider.CacheLookup` span is created for non-finalized methods, and
finalized methods produce a span with `cache.hit=false` (no error).

### Same-machine deployments

If consumer and provider run on the same host, **do not** set
`OTEL_SERVICE_NAME` in a shared environment file. Each binary's in-code
default (`lava-consumer` / `lava-provider`) gives them distinct service
identities. For per-deployment labelling without overriding `service.name`:

```bash
OTEL_RESOURCE_ATTRIBUTES=region=eu-west-1,deployment=prod-1
```

### Sampling

Defaults to `parentbased_always_on`. For high-volume providers, tune via:

```bash
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1   # 10% of root spans
```

The `parentbased_*` family honors the consumer's sampling decision when a
trace originates from a traced consumer relay, so consumer↔provider sampling
stays consistent.
