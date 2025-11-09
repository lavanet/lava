# Compression Not Working - Investigation Guide

## Problem

Consumer log shows:
```json
{
  "compressionEnabled": "true",
  "responseActuallyCompressed": "false",
  "responseSize": "10325421"
}
```

This means:
- Consumer requested compression (`grpc-encoding: gzip` header sent)
- Provider did NOT compress the response (no `grpc-encoding: gzip` in response headers)
- 10MB response was sent uncompressed

## Root Cause Analysis

### Likely Cause: grpc-web + h2c Wrapper Interference

The provider uses a **layered HTTP setup**:
```
HTTP Server (h2c)
  └─> grpc-web Wrapper
      └─> gRPC Server
```

**The issue:** The `grpc-web` wrapper and `h2c` handler may be intercepting/stripping gRPC compression headers before they reach the native gRPC server.

### Evidence

**Location:** `protocol/rpcprovider/provider_listener.go:70-86`

```go
grpcServer := grpc.NewServer(opts...)
wrappedServer := grpcweb.WrapServer(grpcServer)  // <-- Wrapper 1
handler := func(resp http.ResponseWriter, req *http.Request) {
    wrappedServer.ServeHTTP(resp, req)
}
pl.httpServer = http.Server{
    Handler: h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{}),  // <-- Wrapper 2
}
```

## Why Compression Isn't Working

### Theory 1: grpc-web Doesn't Support Native gRPC Compression

`grpcweb.WrapServer` is designed for browser clients and may:
1. Not forward `grpc-encoding` headers to the underlying gRPC server
2. Use its own compression mechanism (if any)
3. Only support unary calls without compression

### Theory 2: h2c Handler Strips Compression Headers

The `h2c.NewHandler` wrapper (HTTP/2 Cleartext) may:
1. Process HTTP/2 frames before gRPC sees them
2. Strip or modify compression-related headers
3. Not properly propagate gRPC-specific metadata

### Theory 3: Response Path Doesn't Preserve Compression

Even if the request reaches gRPC with compression header:
1. gRPC server compresses the response
2. grpc-web wrapper decompresses it when translating to HTTP/1.1 format
3. Response reaches consumer uncompressed

## Diagnostic Steps

### Step 1: Check Provider Logs

Look for these logs in provider output:
```
[INFO] Got relay request from consumer
  requestActuallyCompressed: true/false
  grpcEncoding: [gzip] or []
  acceptEncoding: "gzip, identity"
```

**If `requestActuallyCompressed=false`:** The wrapper is stripping the compression header before it reaches the gRPC handler.

**If `requestActuallyCompressed=true`:** The request compression works, but response compression doesn't.

### Step 2: Add More Debug Logging

Check what provider is actually sending:

**Provider side - after generating response:**
```go
// In rpcprovider_server.go, before returning reply
if err == nil && reply != nil {
    // Try to set compression explicitly via grpc.SetHeader
    grpc.SetHeader(ctx, metadata.Pairs("grpc-encoding", "gzip"))
}
```

### Step 3: Check if Direct gRPC Works

Test if compression works with a **direct gRPC connection** (bypassing grpc-web and h2c):

Create a test client that connects directly to the gRPC port without HTTP wrappers.

## Solutions

### Solution 1: Use Native gRPC Without Wrappers (Recommended)

**For pure gRPC clients** (non-browser), bypass grpc-web:

```go
// Separate listener for native gRPC
grpcOnlyServer := grpc.NewServer(opts...)
pairingtypes.RegisterRelayerServer(grpcOnlyServer, relayServer)
go grpcOnlyServer.Serve(lis)  // No wrappers!
```

This would give you two ports:
- One for browser clients (grpc-web + h2c)
- One for native gRPC clients (with working compression)

### Solution 2: Implement Compression at Application Level

Since gRPC compression is being blocked by wrappers, compress manually:

**Provider side:**
```go
if requestCompressed {
    // Compress reply.Data manually using gzip
    var buf bytes.Buffer
    writer := gzip.NewWriter(&buf)
    writer.Write(reply.Data)
    writer.Close()
    reply.Data = buf.Bytes()
    // Add custom header to indicate manual compression
    grpc.SetHeader(ctx, metadata.Pairs("lava-manual-compression", "gzip"))
}
```

**Consumer side:**
```go
if header.Get("lava-manual-compression")[0] == "gzip" {
    // Decompress reply.Data manually
    reader, _ := gzip.NewReader(bytes.NewReader(reply.Data))
    decompressed, _ := io.ReadAll(reader)
    reply.Data = decompressed
}
```

### Solution 3: Use Different Transport

Consider using:
- **WebSocket** for browser clients (can support compression)
- **REST with gzip Content-Encoding** (standard HTTP compression)
- **Separate pure gRPC endpoint** for SDK/CLI clients

### Solution 4: Check grpc-web Configuration

Some grpc-web implementations support compression:
```go
wrappedServer := grpcweb.WrapServer(grpcServer, 
    grpcweb.WithCompression(true),  // If supported
)
```

## Testing Recommendations

### Test 1: Verify Request Compression Works

Check provider logs:
```bash
# Should show true if consumer compression works
grep "requestActuallyCompressed" provider.log
```

### Test 2: Capture Network Traffic

Use `tcpdump` or Wireshark to see actual wire format:
```bash
sudo tcpdump -i any -s 0 -w grpc.pcap port 2224
```

Look for:
- `grpc-encoding: gzip` in request headers
- `grpc-encoding: gzip` in response headers (likely missing)
- Actual payload size on wire

### Test 3: Direct gRPC Test

Create a simple test client using pure gRPC (no HTTP):
```go
conn, _ := grpc.Dial("provider:2224", 
    grpc.WithTransportCredentials(creds),
    grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
)
// Test if compression works without wrappers
```

## Current Status

✅ **Request compression:** Working (consumer → provider)
❌ **Response compression:** NOT working (provider → consumer)

**Why:** The grpc-web + h2c wrapper stack is likely:
1. Processing the response after gRPC compresses it
2. Converting it to a different format
3. Stripping the `grpc-encoding` header
4. Sending uncompressed data to consumer

## Recommended Next Steps

1. **Add provider-side debug logging** to see if response has `grpc-encoding` header set
2. **Check provider logs** for `requestActuallyCompressed` value
3. **Consider Solution 1** (separate native gRPC endpoint) if consumers are SDK/CLI only
4. **Consider Solution 2** (manual compression) as a workaround
5. **Investigate grpc-web library** for compression support options

## Performance Impact

With 10MB uncompressed responses:
- **Current:** ~10MB over network per request
- **With compression:** ~2MB over network (80% reduction)
- **Wasted bandwidth:** ~8MB per request
- **Wasted time:** ~640ms on 100Mbps link

For your 100 concurrent requests scenario:
- **Total waste:** 800MB bandwidth
- **Total time waste:** ~64 seconds in network transfer

**This is significant!** Fixing compression would have major impact on performance.

