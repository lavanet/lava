# Application-Level Compression Logging

This document describes the application-level compression logging added to help monitor and debug compression between consumers and providers.

## Overview

When the `--allow-grpc-compression-for-consumer-provider-communication` flag is enabled, these logs will show:
- Whether compression is enabled on connections
- Request and response sizes (compressed and uncompressed)
- **Whether responses were ACTUALLY compressed** (detected via `lava-compression` header)
- Compression ratios and bandwidth savings
- Consumer compression support signaling (via `lava-compression-support` header)

**Important:** This uses **application-level compression** (manual gzip in the application), not gRPC's native compression. This was necessary because the `grpc-web` wrapper strips gRPC compression headers.

## Log Locations

### 1. Consumer Side

#### Connection Establishment
**Location:** `protocol/lavasession/consumer_types.go:ConnectRawClientWithTimeout()`

**Logs:**
```
INFO: Establishing gRPC connection to provider
  - providerAddress: <address>
  - compressionEnabled: true/false

INFO: Successfully connected to provider
  - providerAddress: <address>
  - compressionEnabled: true/false
```

#### Sending Request
**Location:** `protocol/rpcconsumer/rpcconsumer_server.go:relayInner()`

**Logs:**
```
INFO: Sending relay to provider <name>
  - GUID: <guid>
  - providerName: <address>
  - requestSize: <bytes>                    # Uncompressed request size
  - compressionFlagEnabled: true/false      # Is flag enabled
  - requestWillBeCompressed: true/false     # Will request be compressed (flag + size > 0)
  - requestBlock: <block>
  - seenBlock: <block>
  - extensions: <extensions>
```

#### Receiving Response
**Location:** `protocol/rpcconsumer/rpcconsumer_server.go:relayInner()`

**Logs:**
```
INFO: Received relay response from provider
  - GUID: <guid>
  - providerName: <address>
  - responseSize: <bytes>                       # Response size (compressed if appLevelCompressed=true)
  - compressionFlagEnabled: true/false          # Is flag enabled
  - grpcCompressed: false                       # gRPC native compression (always false due to grpc-web wrapper)
  - appLevelCompressed: true/false              # Was response compressed at application level
  - relayLatency: <duration>

INFO: Response decompressed
  - GUID: <guid>
  - providerName: <address>
  - compressedSize: <bytes>                     # Size on wire
  - decompressedSize: <bytes>                   # Size after decompression
  - compressionRatio: <percentage>              # Bandwidth saved (e.g., 78.5% means 78.5% reduction)

WARN: Compression was enabled but response was not compressed  # Only if mismatch detected
  - providerName: <address>
  - responseSize: <bytes>
```

### 2. Provider Side

#### Receiving Request
**Location:** `protocol/rpcprovider/rpcprovider_server.go:Relay()`

**Logs:**
```
INFO: Got relay request from consumer
  - GUID: <guid>
  - request.path: <api_path>
  - request.SessionId: <session_id>
  - requestSize: <bytes>                            # Request size (always small)
  - requestActuallyCompressed: false                # Requests are never compressed (always small)
  - grpcEncoding: []                                # gRPC compression (not used due to grpc-web wrapper)
  - acceptEncoding: "gzip, identity, ..."           # gRPC automatic header (not reliable)
  - consumerSupportsCompression: true/false         # Custom header indicating consumer flag is enabled
  - request.cu: <compute_units>
  - relay_timeout: <timeout>
```

#### Sending Response
**Location:** `protocol/rpcprovider/rpcprovider_server.go:Relay()`

**Logs:**
```
INFO: Response compressed
  - GUID: <guid>
  - originalSize: <bytes>                           # Size before compression
  - compressedSize: <bytes>                         # Size after compression
  - compressionRatio: <percentage>                  # Bandwidth saved
  - savedBytes: <bytes>                             # Bytes saved

INFO: Done handling relay request from consumer
  - GUID: <guid>
  - request.SessionId: <session_id>
  - responseSize: <bytes>                           # Final response size (compressed if responseCompressed=true)
  - responseCompressed: true/false                  # Was response compressed
  - compressionRatio: <percentage>                  # Bandwidth saved (0.0 if not compressed)
  - timeTaken: <duration>
```

## How Compression Works

### Request Flow (Consumer → Provider)
1. Consumer sets `grpc.UseCompressor(gzip.Name)` when `compressionEnabled=true`
2. Consumer compresses request before sending
3. Consumer adds `grpc-encoding: gzip` header
4. Provider receives compressed request
5. Provider detects compression via `grpc-encoding` header
6. Provider automatically decompresses request

### Response Flow (Provider → Consumer)
1. Provider checks if request had `grpc-encoding: gzip` header
2. If yes, provider automatically compresses response
3. Provider sends compressed response
4. Consumer automatically decompresses response

**Note:** Compression is bidirectional but client-initiated. The provider only compresses responses when the consumer requests it via the `grpc-encoding` header.

## Example Log Output

### With Compression Enabled

**Consumer:**
```
[INFO] Establishing gRPC connection to provider providerAddress=provider1.example.com:2224 compressionEnabled=true
[INFO] Successfully connected to provider providerAddress=provider1.example.com:2224 compressionEnabled=true
[INFO] Sending relay to provider provider1 requestSize=2048 compressionFlagEnabled=true requestWillBeCompressed=true
[INFO] Received relay response from provider providerName=provider1 responseSize=15728640 compressionFlagEnabled=true responseActuallyCompressed=true relayLatency=8.5s
```

**Provider:**
```
[INFO] Got relay request from consumer request.path=/eth/v1/blocks/latest requestSize=2048 requestActuallyCompressed=true grpcEncoding=[gzip] acceptEncoding="gzip, identity"
[INFO] Done handling relay request from consumer responseSize=15728640 willCompressResponse=true timeTaken=8.2s
```

### Without Compression

**Consumer:**
```
[INFO] Establishing gRPC connection to provider providerAddress=provider1.example.com:2224 compressionEnabled=false
[INFO] Successfully connected to provider providerAddress=provider1.example.com:2224 compressionEnabled=false
[INFO] Sending relay to provider provider1 requestSize=2048 compressionFlagEnabled=false requestWillBeCompressed=false
[INFO] Received relay response from provider providerName=provider1 responseSize=15728640 compressionFlagEnabled=false responseActuallyCompressed=false relayLatency=11.2s
```

**Provider:**
```
[INFO] Got relay request from consumer request.path=/eth/v1/blocks/latest requestSize=2048 requestActuallyCompressed=false grpcEncoding=[] acceptEncoding=""
[INFO] Done handling relay request from consumer responseSize=15728640 willCompressResponse=false timeTaken=8.1s
```

### Mismatch Scenario (Compression Enabled but Not Working)

**Consumer:**
```
[INFO] Sending relay to provider provider1 requestSize=2048 compressionFlagEnabled=true requestWillBeCompressed=true
[INFO] Received relay response from provider providerName=provider1 responseSize=15728640 compressionFlagEnabled=true responseActuallyCompressed=false relayLatency=10.8s
[WARN] Compression was enabled but response was not compressed providerName=provider1 responseSize=15728640
```

This indicates a configuration or compatibility issue where compression was requested but not applied.

## Analyzing Compression Impact

### What to Look For

1. **Verify Actual Compression:**
   - Check `requestActuallyCompressed=true` on provider side
   - Check `responseActuallyCompressed=true` on consumer side
   - Check `grpcEncoding=[gzip]` is present

2. **Request Compression Overhead:**
   - Compare `requestSize` with transfer time
   - Small requests (< 10 KB) may show minimal benefit
   - Look for `requestActuallyCompressed=true` to confirm it happened

3. **Response Compression Benefit:**
   - Compare `relayLatency` with compression enabled vs disabled
   - Large responses (> 1 MB) should show significant improvement
   - Expected savings: 70-85% for JSON data
   - Verify `responseActuallyCompressed=true`

4. **Compression Detection:**
   - `requestActuallyCompressed=true` on provider confirms consumer compressed the request
   - `responseActuallyCompressed=true` on consumer confirms provider compressed the response
   - `grpcEncoding=[gzip]` shows the actual header value
   - Mismatch triggers a WARNING log

### Performance Metrics to Track

```bash
# Calculate bandwidth saved
uncompressed_size * 0.75 = approximate_bytes_saved

# Network time saved (assuming 100 Mbps)
bytes_saved * 8 / 100000000 = seconds_saved

# Check if latency improved
latency_without_compression - latency_with_compression = net_benefit
```

## Troubleshooting

### Issue: `requestActuallyCompressed=false` on provider but compression enabled on consumer

**Possible Causes:**
1. Consumer flag not actually enabled - check consumer startup logs
2. Connection established before flag was set
3. gRPC compression not supported by client library version
4. Empty request body (size=0)

**Solution:**
- Restart consumer with flag enabled
- Check consumer connection logs for `compressionEnabled=true`
- Verify `grpcEncoding=[gzip]` in provider logs
- Check that `requestSize > 0`

### Issue: `responseActuallyCompressed=false` but flag enabled

**Possible Causes:**
1. Provider's gRPC server doesn't support compression
2. Request didn't include `grpc-encoding: gzip` header
3. Provider gRPC library version issue

**Solution:**
- Check provider logs for `requestActuallyCompressed=true`
- Verify `grpcEncoding=[gzip]` in provider request logs
- Check consumer sends `grpc-encoding` header
- WARNING log will alert you to this mismatch

### Issue: Large responses but no latency improvement

**Possible Causes:**
1. CPU bottleneck - compression time > network time saved
2. Local network (< 10ms latency) - not enough network time to save
3. Compression not actually happening - check `requestCompressed` field

**Solution:**
- Check CPU usage during requests
- Verify `requestActuallyCompressed=true` and `responseActuallyCompressed=true`
- Compare actual network transfer times
- Look for `grpcEncoding=[gzip]` to confirm wire-level compression

### Issue: Increased latency with compression

**Possible Causes:**
1. Small payloads - compression overhead exceeds benefit
2. CPU-constrained environment
3. Already compressed data (images, pre-compressed JSON)

**Solution:**
- Only enable for large response APIs
- Check response data type - may not compress well
- Disable if responses are typically < 100 KB

## Configuration

To enable compression logging, run:

**Consumer:**
```bash
lavad rpcconsumer \
  --from alice \
  --geolocation 1 \
  --allow-grpc-compression-for-consumer-provider-communication
```

**Provider:**
No special flag needed - provider logs will automatically show compression status based on incoming request headers.

## Summary

These logs provide complete visibility into:
- ✅ Whether compression is actually enabled and working (not just configured)
- ✅ **Actual wire-level compression detection** via `grpc-encoding` headers
- ✅ Request and response sizes (uncompressed, for calculating savings)
- ✅ Impact on latency
- ✅ Which side is compressing what (detected, not assumed)
- ✅ Automatic warning when compression is expected but not happening

**Key Features:**
- Detects compression via `grpc-encoding: gzip` header on both request and response
- Shows actual `grpcEncoding` and `acceptEncoding` header values
- Warns when configuration and reality don't match
- All sizes are uncompressed (after decompression) for accurate comparison

Use this data to make informed decisions about whether compression benefits your specific workload.

