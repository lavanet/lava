# Provider Selection Header Implementation

## Overview
This feature allows users to select a specific provider to relay their requests through by using a custom header. This functionality is **only available in smartrouter** and is **not implemented in rpcconsumer**.

## Header Name
```
lava-select-provider
```

## Usage

### Request Example
When making a request to smartrouter, include the header with the provider's address:

```bash
curl -X POST http://smartrouter-endpoint \
  -H "Content-Type: application/json" \
  -H "lava-select-provider: lava@1abc2def3ghi4jkl5mno6pqr7stu8vwx9yz0" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### How It Works

1. **Header Parsing**: The smartrouter extracts the `lava-select-provider` header from incoming requests
2. **Provider Validation**: The selected provider is validated against the list of valid providers for the endpoint (considering addons and extensions)
3. **Provider Selection**: If valid, the specified provider is used exclusively for that request
4. **Error on Invalid Provider**: If the provider is invalid (not in the valid providers list), the request **fails with an error** instead of falling back to random selection

## Implementation Details

### Modified Files

1. **`protocol/common/endpoints.go`**
   - Added `SELECT_PROVIDER_HEADER_NAME = "lava-select-provider"` constant
   - Registered it in `SPECIAL_LAVA_DIRECTIVE_HEADERS` map

2. **`protocol/lavasession/consumer_session_manager.go`**
   - Modified `GetSessions()` signature to accept `selectedProvider string` parameter
   - Modified `getSessionWithProviderOrError()` to pass through the parameter
   - Modified `getValidConsumerSessionsWithProvider()` to pass through the parameter
   - Modified `getValidProviderAddresses()` to implement provider selection logic:
     - Checks if `selectedProvider` is not empty
     - Validates the provider is in the valid addresses list
     - Returns the selected provider if valid
     - Falls back to normal selection if invalid

3. **`protocol/rpcsmartrouter/rpcsmartrouter_server.go`**
   - Extracts the `lava-select-provider` header from directive headers
   - Passes the selected provider to `GetSessions()`
   - Logs provider selection for debugging

4. **`protocol/rpcconsumer/rpcconsumer_server.go`**
   - Updated `GetSessions()` calls to pass empty string `""` for `selectedProvider`
   - No behavior change - feature not enabled for rpcconsumer

5. **Test files**
   - Updated all test calls to `GetSessions()` to include the new parameter

### Provider Validation

The selected provider must be:
- In the current valid provider list for the endpoint
- Support the requested addons (if any)
- Support the requested extensions (if any)

If the provider doesn't meet these criteria, **the request fails with an error**:
```
Selected provider not available | selectedProvider=lava@invalid... | validProviders=[...] | addon=... | extensions=[...]
```

### Logging and Error Messages

**Success - Provider selected via header:**
```
Provider selected via header | provider=lava@1abc... | addon=archive | extensions=[archive]
```

**Error - Invalid provider specified:**
```
Error: Selected provider not available
selectedProvider=lava@invalid...
validProviders=[lava@provider1, lava@provider2, ...]
addon=archive
extensions=[archive]
```

The client will receive an HTTP error response (typically 500) with details about why the provider selection failed.

## Behavior Differences: Smartrouter vs RPCConsumer

| Feature | Smartrouter | RPCConsumer |
|---------|------------|-------------|
| Provider Selection Header | ✅ Supported | ❌ Not Supported |
| Header Processing | Extracts and uses | Ignores (passes empty string) |
| Use Case | Centralized routing with known providers | Decentralized pairing via blockchain |

## Security Considerations

- The header only works for providers that are already configured/paired with the smartrouter
- Invalid provider addresses are rejected with a warning
- The feature does not bypass provider validation or pairing logic
- Provider selection still respects addons and extensions requirements

## Testing

To test this feature:

1. **Valid Provider Test:**
   ```bash
   curl -i -X POST http://smartrouter:3360 \
     -H "lava-select-provider: primary-2222-jsonrpc" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
   ```
   ✅ Expected: Response header `Lava-Provider-Address: primary-2222-jsonrpc`

2. **Invalid Provider Test:**
   ```bash
   curl -i -X POST http://smartrouter:3360 \
     -H "lava-select-provider: invalid-provider" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
   ```
   ❌ Expected: HTTP error response with message "Selected provider not available"

3. **Provider Without Extension Support:**
   ```bash
   curl -i -X POST http://smartrouter:3360 \
     -H "lava-select-provider: provider-without-archive" \
     -H "lava-extension: archive" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x1",true],"id":1}'
   ```
   ❌ Expected: HTTP error response with message "Selected provider not available"

## Example Use Cases

1. **Debugging**: Route requests to a specific provider to test its behavior
2. **Testing**: Verify a newly added provider works correctly
3. **Performance Testing**: Compare response times from different providers
4. **Provider Affinity**: Stick to a specific provider for troubleshooting
