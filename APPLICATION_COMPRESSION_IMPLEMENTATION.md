# Application-Level Compression Implementation

## Summary

This document describes the custom application-level compression implementation that bypasses gRPC's native compression due to the `grpc-web` wrapper stripping compression headers.

## Problem

The `grpc-web` wrapper and `h2c` handler in the provider's gRPC server setup were stripping the `grpc-encoding: gzip` header from native gRPC requests. This prevented gRPC's built-in compression from working, even when the flag was enabled.

## Solution

Implement **application-level compression** that:
1. Uses a custom header (`lava-compression-support`) to signal compression support
2. Manually compresses/decompresses payloads at the application layer
3. Uses size-based logic to only compress large payloads (>10 KB)
4. Tracks compression ratios and bandwidth savings

## Implementation Details

### 1. Custom Headers

**`lava-compression-support: true`** (Consumer → Provider)
- Sent by consumer when the `--allow-grpc-compression-for-consumer-provider-communication` flag is enabled
- Signals to provider that consumer supports receiving compressed responses
- Different from `grpc-accept-encoding` which is always present (sent automatically by gRPC library)

**`lava-compression: gzip`** (Provider → Consumer)
- Sent by provider when a response is compressed
- Tells consumer to decompress the response payload

### 2. Compression Logic

**Location:** `protocol/common/compression.go`

**Key Functions:**
- `CompressData(data []byte, threshold int) ([]byte, bool, error)`
  - Only compresses if payload > threshold (10 KB default)
  - Uses `gzip.BestSpeed` for fast compression
  - Only returns compressed data if it's actually smaller
  
- `DecompressData(data []byte) ([]byte, error)`
  - Decompresses gzip-compressed data
  
- `CalculateCompressionRatio(originalSize, compressedSize int) float64`
  - Returns percentage saved (e.g., 78.5% = 78.5% reduction)

**Key Constants:**
- `CompressionThreshold = 10 * 1024` (10 KB)
- `CompressionLevel = gzip.BestSpeed` (prioritizes speed over ratio)
- `LavaCompressionSupportHeader = "lava-compression-support"`
- `LavaCompressionHeader = "lava-compression"`
- `LavaCompressionGzip = "gzip"`

### 3. Consumer Side

**File:** `protocol/rpcconsumer/rpcconsumer_server.go`

**Changes:**
1. **Send compression support header** when flag is enabled:
   ```go
   if compressionEnabled {
       metadataAdd.Set(common.LavaCompressionSupportHeader, "true")
   }
   ```

2. **Check for compressed response** via custom header:
   ```go
   appLevelCompressed := false
   if lavaCompressionValues := responseHeader.Get(common.LavaCompressionHeader); len(lavaCompressionValues) > 0 {
       appLevelCompressed = lavaCompressionValues[0] == common.LavaCompressionGzip
   }
   ```

3. **Decompress response** if needed:
   ```go
   if reply != nil && reply.Data != nil && appLevelCompressed {
       decompressedData, decompressErr := common.DecompressData(reply.Data)
       // ... handle errors and log metrics
       reply.Data = decompressedData
   }
   ```

4. **Log metrics**:
   - `appLevelCompressed`: Whether response was compressed
   - `compressionRatio`: Percentage of bandwidth saved
   - `savedBytes`: Actual bytes saved

### 4. Provider Side

**File:** `protocol/rpcprovider/rpcprovider_server.go`

**Changes:**
1. **Check consumer compression support** via custom header:
   ```go
   compressionSupport := md.Get(common.LavaCompressionSupportHeader)
   consumerSupportsCompression := len(compressionSupport) > 0 && compressionSupport[0] == "true"
   ```

2. **Compress response** if consumer supports it:
   ```go
   if consumerSupportsCompression {
       compressedData, wasCompressed, compressErr := common.CompressData(reply.Data, common.CompressionThreshold)
       if wasCompressed {
           reply.Data = compressedData
           grpc.SetHeader(ctx, metadata.Pairs(common.LavaCompressionHeader, common.LavaCompressionGzip))
       }
   }
   ```

3. **Log metrics**:
   - `consumerSupportsCompression`: Whether consumer sent support header
   - `responseCompressed`: Whether response was compressed
   - `compressionRatio`: Percentage saved
   - `savedBytes`: Actual bytes saved

## Logging

### Consumer Logs

**Sending request:**
```
INFO: Sending relay to provider
  - requestSize: 83
  - compressionFlagEnabled: true
```

**Receiving response (compressed):**
```
INFO: Response decompressed
  - compressedSize: 4325421
  - decompressedSize: 16964805
  - compressionRatio: 74.51%
  - savedBytes: 12639384

INFO: Received relay response from provider
  - responseSize: 16964805
  - compressionFlagEnabled: true
  - appLevelCompressed: true
  - relayLatency: 8.5s
```

**Receiving response (not compressed due to size):**
```
INFO: Received relay response from provider
  - responseSize: 8192
  - compressionFlagEnabled: true
  - appLevelCompressed: false
  - relayLatency: 0.2s
```

### Provider Logs

**Receiving request:**
```
INFO: Got relay request from consumer
  - requestSize: 83
  - consumerSupportsCompression: true
```

**Sending response (compressed):**
```
INFO: Response compressed
  - originalSize: 16964805
  - compressedSize: 4325421
  - compressionRatio: 74.51%
  - savedBytes: 12639384

INFO: Done handling relay request from consumer
  - responseSize: 4325421
  - responseCompressed: true
  - compressionRatio: 74.51%
```

**Sending response (not compressed due to size):**
```
INFO: Done handling relay request from consumer
  - responseSize: 8192
  - responseCompressed: false
  - compressionRatio: 0.0
```

## Why This Approach?

### Why Not gRPC Native Compression?

**Problem:** The `grpc-web` wrapper strips `grpc-encoding` headers, preventing native gRPC compression from working.

**Evidence:**
- Consumer logs showed `compressionFlagEnabled: true` but `grpcEncoding: []` on provider
- Provider never received `grpc-encoding: gzip` header from consumer
- Provider couldn't respond with compression because it didn't know consumer supported it

### Why Custom Headers?

**Problem:** `grpc-accept-encoding` header is **always present** (sent automatically by gRPC client library), making it unreliable for detecting our flag.

**Solution:** Use `lava-compression-support` custom header that's **only sent when flag is enabled**.

### Why Size-Based Logic?

**Reason:** Small payloads have compression overhead > bandwidth savings.

**Implementation:**
- Only compress if payload > 10 KB
- Only use compressed version if it's actually smaller
- Skip compression entirely for small responses

## Benefits

1. ✅ **Works despite grpc-web wrapper** - bypasses the header stripping issue
2. ✅ **Size-based compression** - only compresses when beneficial
3. ✅ **Explicit control** - only compresses when flag is enabled (not based on unreliable headers)
4. ✅ **Detailed metrics** - logs compression ratio and bytes saved
5. ✅ **Fast compression** - uses `gzip.BestSpeed` for minimal CPU overhead
6. ✅ **Backward compatible** - consumers without flag work normally

## Performance Impact

**Expected compression ratios for JSON data:**
- Small responses (< 10 KB): Not compressed (overhead > benefit)
- Medium responses (10-100 KB): 60-70% reduction
- Large responses (> 1 MB): 70-85% reduction

**CPU overhead:**
- Compression: ~5-10ms per MB using `gzip.BestSpeed`
- Decompression: ~2-5ms per MB

**Network time saved (100 Mbps connection):**
- 10 MB response @ 75% compression = 7.5 MB saved = ~600ms faster
- 1 MB response @ 70% compression = 700 KB saved = ~56ms faster

## Testing

To test the implementation:

1. **Enable flag on consumer:**
   ```bash
   lavad rpcconsumer --allow-grpc-compression-for-consumer-provider-communication
   ```

2. **Make requests with large responses** (> 10 KB)

3. **Check logs for:**
   - Consumer: `appLevelCompressed: true`
   - Provider: `consumerSupportsCompression: true` and `responseCompressed: true`
   - Compression ratio and bytes saved

4. **Verify bandwidth savings:**
   - Compare `compressedSize` vs `decompressedSize`
   - Check `compressionRatio` percentage
   - Monitor `relayLatency` improvement

## Configuration

**Threshold adjustment** (if needed):

Edit `protocol/common/compression.go`:
```go
const CompressionThreshold = 10 * 1024  // Change to desired size in bytes
```

**Compression level adjustment** (if needed):
```go
const CompressionLevel = gzip.BestSpeed  // Options: BestSpeed (1), BestCompression (9), DefaultCompression (6)
```

## Troubleshooting

### Issue: `appLevelCompressed: true` even though flag is disabled

**Cause:** This should not happen with the custom header approach.

**Check:**
- Provider logs should show `consumerSupportsCompression: false` if flag is disabled
- If you see `consumerSupportsCompression: true`, the flag is actually enabled

### Issue: `appLevelCompressed: false` but flag is enabled and response is large

**Possible causes:**
1. Response size < threshold (10 KB) - this is expected
2. Compressed size >= original size - compression didn't help, so uncompressed version is used
3. Compression error - check for warning logs

**Check logs for:**
- `responseSize` < 10240 (below threshold)
- Provider: `responseCompressed: false` and `compressionRatio: 0.0`

### Issue: Increased latency with compression

**Possible causes:**
1. Very fast network (< 10ms) - compression time > network time saved
2. CPU-constrained environment
3. Data doesn't compress well

**Solutions:**
- Increase threshold: `CompressionThreshold = 50 * 1024` (50 KB)
- Use faster compression: Already using `gzip.BestSpeed`
- Disable flag for fast/local networks

