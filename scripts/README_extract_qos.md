# QoS Extraction Script

## Overview

`extract_qos_from_consumer_log.py` extracts non-normalized QoS (Quality of Service) metrics from consumer logs for analysis.

## What It Extracts

For each request in the consumer log, the script extracts:

- **Request ID**: Sequential request number
- **Timestamp**: When the provider selection occurred
- **GUID**: Request GUID (if available in the log)
- **Providers**: For each provider considered:
  - `latency`: Raw latency value (in seconds)
  - `availability`: Raw availability score (0-1)
  - `sync`: Raw sync lag (in seconds)
  - `stake`: Provider stake amount (without "ulava" suffix)
- **Selected Provider**: Which provider was chosen for the request

## Usage

### Basic Usage

```bash
# Output to console
python3 scripts/extract_qos_from_consumer_log.py testutil/debugging/logs/CONSUMER.log

# Save to JSON file
python3 scripts/extract_qos_from_consumer_log.py testutil/debugging/logs/CONSUMER.log output.json
```

### Example Output

```json
[
  {
    "request_id": 1,
    "timestamp": "Feb  4 15:59:16",
    "guid": null,
    "providers": {
      "primary-2220-jsonrpc": {
        "provider": "primary-2220-jsonrpc",
        "latency": "0.00093708",
        "availability": "1",
        "sync": "0.1",
        "stake": "10"
      },
      "primary-2221-jsonrpc": {
        "provider": "primary-2221-jsonrpc",
        "latency": "1e-08",
        "availability": "1",
        "sync": "0.1",
        "stake": "10"
      },
      "primary-2222-jsonrpc": {
        "provider": "primary-2222-jsonrpc",
        "latency": "1e-08",
        "availability": "1",
        "sync": "0.1",
        "stake": "10"
      }
    },
    "selected_provider": "primary-2222-jsonrpc"
  }
]
```

## Analyzing the Output

You can use Python to analyze the extracted data:

```python
import json
from collections import Counter

with open('qos_output.json') as f:
    data = json.load(f)

# Count selected providers
selected = Counter(req['selected_provider'] for req in data)
print('Provider selection distribution:')
for provider, count in selected.most_common():
    percentage = count / len(data) * 100
    print(f'{provider}: {count} ({percentage:.1f}%)')

# Analyze latency for a specific provider
latencies = []
for req in data:
    if 'primary-2220-jsonrpc' in req['providers']:
        latency = float(req['providers']['primary-2220-jsonrpc']['latency'])
        latencies.append(latency)

avg_latency = sum(latencies) / len(latencies)
print(f'Average latency for primary-2220-jsonrpc: {avg_latency:.6f}s')
```

## How It Works

The script processes the consumer log sequentially:

1. **Parses "Provider score calculation breakdown" logs**: These contain the raw (not normalized) QoS values:
   - `raw_latency_sec`
   - `raw_availability`
   - `raw_sync_sec`
   - `raw_stake`

2. **Accumulates provider data**: As it encounters score breakdown logs, it accumulates QoS data for each provider.

3. **Detects "Provider selection completed" logs**: When a selection is made, it:
   - Captures all the accumulated provider data
   - Records which provider was selected
   - Creates a request entry in the output
   - Clears the accumulated data for the next request

## Notes

- The script handles very large log files efficiently by processing line-by-line
- Invalid UTF-8 characters in the log are ignored
- Requests without a selected provider are filtered out
- The `stake` value has the "ulava" suffix removed for cleaner output
- Some requests may have 0 providers if no score breakdown was logged before selection (typically 375 out of 10,876 requests)

## Quick Analysis

After extracting QoS data, you can use the companion analysis script:

```bash
python3 scripts/analyze_qos_output.py qos_output.json
```

This will provide:
- Provider selection frequency and distribution
- Average QoS metrics per provider
- Competitive selection analysis (2-way vs 3-way competitions)
- Interesting edge cases where lowest latency wasn't selected

## Integration with Analysis Scripts

This script complements:
- `scripts/analyze_qos_output.py` - Quick statistical analysis and summaries
- `scripts/analyze_wrs.py` - Detailed weighted random selection analysis with visualizations
