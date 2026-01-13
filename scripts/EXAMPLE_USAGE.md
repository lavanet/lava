# QoS Extraction - Example Usage

This guide demonstrates the complete workflow for extracting and analyzing QoS data from consumer logs.

## Step 1: Extract QoS Data

```bash
# Extract QoS data from consumer log
python3 scripts/extract_qos_from_consumer_log.py \
    testutil/debugging/logs/CONSUMER.log \
    qos_output.json
```

**Output:**
```
Processing log file: testutil/debugging/logs/CONSUMER.log
Extracted 772 requests with complete QoS data

============================================================
QoS EXTRACTION SUMMARY
============================================================

Total requests: 772
Timestamp range: Feb  4 18:27:13 to Feb  4 18:27:54

Selected provider distribution:
  primary-2222-jsonrpc          :    291 ( 37.7%)
  primary-2221-jsonrpc          :    245 ( 31.7%)
  primary-2220-jsonrpc          :    236 ( 30.6%)

Providers evaluated per request:
  0 providers:     98 requests ( 12.7%)
  1 provider :     74 requests (  9.6%)
  2 providers:    311 requests ( 40.3%)
  3 providers:    289 requests ( 37.4%)

Output written to: qos_output.json
```

## Step 2: Analyze Extracted Data

```bash
# Run analysis on extracted data
python3 scripts/analyze_qos_output.py qos_output.json
```

**Key Insights:**
- Provider selection is fairly distributed (~30-37% each)
- Most requests (40%) had 2 competing providers
- 37% had full 3-way competition
- Selection is probabilistic (weighted random), so lower latency doesn't always win

## Step 3: Inspect Individual Requests

```bash
# View first 2 requests in detail
python3 -c "
import json
with open('qos_output.json') as f:
    data = json.load(f)
    print(json.dumps(data[0:2], indent=2))
"
```

**Sample Output:**
```json
[
  {
    "request_id": 1,
    "timestamp": "Feb  4 18:27:13",
    "guid": null,
    "providers": {
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
      },
      "primary-2220-jsonrpc": {
        "provider": "primary-2220-jsonrpc",
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

## Step 4: Custom Analysis

Create custom analysis scripts using the extracted data:

```python
import json
from collections import defaultdict

# Load data
with open('qos_output.json') as f:
    data = json.load(f)

# Count how often each provider wins when it has best latency
best_latency_wins = defaultdict(lambda: {'wins': 0, 'total': 0})

for req in data:
    if len(req['providers']) < 2:
        continue
    
    # Find provider with best (lowest) latency
    latencies = {p: float(qos['latency']) 
                 for p, qos in req['providers'].items()}
    best_provider = min(latencies, key=latencies.get)
    
    best_latency_wins[best_provider]['total'] += 1
    if best_provider == req['selected_provider']:
        best_latency_wins[best_provider]['wins'] += 1

# Print results
for provider, stats in best_latency_wins.items():
    win_rate = stats['wins'] / stats['total'] * 100
    print(f"{provider}: {stats['wins']}/{stats['total']} ({win_rate:.1f}%)")
```

## Understanding the Data

### QoS Metrics

- **latency**: Time in seconds for request/response (raw value)
- **availability**: Provider availability score (0-1, where 1 is always available)
- **sync**: Sync lag in seconds (how far behind the provider is)
- **stake**: Provider's stake amount (without "ulava" suffix)

### Selection Process

The weighted random selector uses these metrics to calculate a composite score for each provider:
- Lower latency → higher score
- Higher availability → higher score
- Lower sync lag → higher score
- Higher stake → higher score

The provider is then selected probabilistically based on these scores.

## Troubleshooting

### No data extracted
- Check that the log file contains "Provider score calculation breakdown" entries
- Verify the log is from a consumer (not provider)
- Ensure the log has "Provider selection completed" entries

### Missing providers in some requests
- This is normal - not all providers may be evaluated for every request
- Providers may be excluded due to blocklisting, connection issues, or addon requirements

### All QoS values are the same
- This can happen during initial testing or warm-up periods
- Check a larger time range of logs for more variation
