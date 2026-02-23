# Consumer Log Parser

Parse CONSUMER.log files and generate structured JSON with provider metrics and selection data.

## Overview

This script extracts provider performance metrics and selection patterns from Lava consumer logs, producing a JSON file suitable for analysis and visualization.

## Features

- **Provider Metrics**: Availability, latency, and sync time series data
- **Provider Stakes**: Extracted from log entries
- **Selection Tracking**: Observed provider selections over time
- **Time Aggregation**: Configurable sampling intervals
- **Optimized Performance**: Processes millions of log lines efficiently

## Usage

### Basic Usage

```bash
python3 scripts/parse_consumer_log.py testutil/debugging/logs/CONSUMER.log
```

This creates `consumer_metrics.json` in the current directory with 60-second sampling intervals.

### Custom Output and Options

```bash
python3 scripts/parse_consumer_log.py \
  testutil/debugging/logs/CONSUMER.log \
  -o output/metrics.json \
  --interval 30000 \
  --pretty
```

### Command-Line Options

- `input_file` (required): Path to CONSUMER.log file
- `-o, --output`: Output JSON file path (default: `consumer_metrics.json`)
- `-i, --interval`: Sampling interval in milliseconds (default: `60000` = 1 minute)
- `--pretty`: Pretty-print JSON output with indentation

### Examples

**1-minute sampling (default):**
```bash
python3 scripts/parse_consumer_log.py testutil/debugging/logs/CONSUMER.log
```

**30-second sampling:**
```bash
python3 scripts/parse_consumer_log.py testutil/debugging/logs/CONSUMER.log --interval 30000
```

**5-minute sampling:**
```bash
python3 scripts/parse_consumer_log.py testutil/debugging/logs/CONSUMER.log --interval 300000
```

## Output Format

The generated JSON follows this structure:

```json
{
  "meta": {
    "schemaVersion": "1.0",
    "timeUnit": "seconds",
    "latencyUnit": "seconds",
    "syncUnit": "seconds",
    "startTs": 1770285446,
    "endTs": 1770313055,
    "sampleIntervalMs": 60000
  },
  "providers": [
    {
      "name": "primary-2220-jsonrpc",
      "stake": 10,
      "series": [
        {
          "ts": 1770285420,
          "availability": 1.0,
          "latency": 0.083672,
          "sync": 0.04
        }
      ]
    }
  ],
  "observedSelections": [
    {
      "ts": 1770285448,
      "provider": "primary-2222-jsonrpc",
      "count": 1
    }
  ]
}
```

### Field Descriptions

#### Meta Section
- `schemaVersion`: Format version (currently "1.0")
- `timeUnit`: Unit for timestamps ("seconds")
- `latencyUnit`: Unit for latency measurements ("seconds")
- `syncUnit`: Unit for sync measurements ("seconds")
- `startTs`: Unix timestamp of first log entry
- `endTs`: Unix timestamp of last log entry
- `sampleIntervalMs`: Sampling interval in milliseconds

#### Provider Data
- `name`: Provider identifier
- `stake`: Provider stake amount (in ulava units)
- `series`: Array of time-series data points
  - `ts`: Unix timestamp (bucket start time)
  - `availability`: Availability score (0.0 to 1.0)
  - `latency`: Average latency in seconds
  - `sync`: Average sync time in seconds

#### Observed Selections
- `ts`: Unix timestamp when selection occurred
- `provider`: Provider that was selected
- `count`: Number of times selected at this timestamp

## Performance

The optimized parser uses:
- Fast string parsing instead of complex regex
- Efficient file I/O with 1MB buffering
- Progress indicators every 100,000 lines

**Typical Performance:**
- ~2 million lines: ~7-8 minutes
- Memory efficient: processes line-by-line
- Output size: ~700KB for typical consumer logs

## Data Extraction

The script extracts data from these log patterns:

1. **Provider Selections**: `"Choosing providers ... chosenProviders=..."`
2. **Latency Probes**: `"[Optimizer] probe update latency=... providerAddress=..."`
3. **Detailed Metrics**: `"Provider score calculation breakdown ... raw_availability=... raw_latency_sec=... raw_stake=... raw_sync_sec=..."`

## Aggregation

Metrics are aggregated into time buckets based on the sampling interval:
- Multiple measurements within a bucket are averaged
- Availability: mean of all availability scores in bucket
- Latency: mean of all latency measurements in bucket
- Sync: mean of all sync measurements in bucket

## Use Cases

- **Performance Analysis**: Track provider latency and availability over time
- **Selection Distribution**: Analyze which providers are chosen and how often
- **Visualization**: Feed data into charts/graphs for monitoring dashboards
- **Testing**: Verify weighted selection algorithms and provider routing logic
- **Debugging**: Identify patterns in provider behavior and selection

## Integration

The output JSON can be used with:
- Data visualization tools (D3.js, Chart.js, etc.)
- Analysis scripts (Python pandas, R, etc.)
- Monitoring dashboards
- Testing frameworks

## Notes

- All timestamps are Unix timestamps (seconds since epoch)
- Latency values are in seconds (not milliseconds or microseconds)
- Stake values are parsed from ulava format (e.g., "10ulava" → 10)
- Availability ranges from 0.0 (completely unavailable) to 1.0 (fully available)
