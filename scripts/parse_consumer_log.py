#!/usr/bin/env python3
"""
Parse CONSUMER.log and generate structured JSON with provider metrics and selections.

This script extracts:
- Provider time series data (availability, latency, sync)
- Provider stake information
- Observed provider selections over time

Optimized for large log files (millions of lines).
"""

import re
import json
import sys
from datetime import datetime
from collections import defaultdict
from typing import Dict, List, Any
import argparse


class ConsumerLogParser:
    """Parser for CONSUMER.log files to extract provider metrics."""

    def __init__(self):
        self.providers = defaultdict(lambda: {
            "name": "",
            "stake": 0,
            "metrics": []  # List of (timestamp, availability, latency, sync)
        })
        self.selections = []  # List of (timestamp, provider, count)
        self.selection_counts = defaultdict(lambda: defaultdict(int))  # ts -> provider -> count
        
        # Timestamp of first and last log entry
        self.start_ts = None
        self.end_ts = None
        
        # Compiled patterns for faster matching
        self.timestamp_pattern = re.compile(r'^([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}(?:\.\d{9})?)')
        
        # Simplified patterns - only compile once
        self.has_choosing = "Choosing providers"
        self.has_probe = "[Optimizer] probe update"
        self.has_score = "Provider score calculation breakdown"

    def parse_timestamp(self, line: str) -> float:
        """Parse timestamp from log line and convert to Unix timestamp with nanosecond precision."""
        match = self.timestamp_pattern.match(line)
        if not match:
            return None
        
        timestamp_str = match.group(1)
        # Parse timestamp (assumes current year 2026)
        try:
            # Check if nanoseconds are present
            if '.' in timestamp_str:
                # Split base time and nanoseconds
                base_time, nanoseconds = timestamp_str.rsplit('.', 1)
                dt = datetime.strptime(f"2026 {base_time}", "%Y %b %d %H:%M:%S")
                # Convert to timestamp and add nanosecond precision
                base_timestamp = dt.timestamp()
                nano_fraction = int(nanoseconds) / 1_000_000_000
                return base_timestamp + nano_fraction
            else:
                # Old format without nanoseconds
                dt = datetime.strptime(f"2026 {timestamp_str}", "%Y %b %d %H:%M:%S")
                return dt.timestamp()
        except ValueError:
            return None

    def parse_latency(self, latency_str: str) -> float:
        """
        Parse latency string to seconds.
        Examples: "460.291µs", "1.514667ms", "0s", "1e-08"
        """
        if latency_str == "0s":
            return 0.0
        
        # Remove quotes if present
        latency_str = latency_str.strip('"')
        
        # Handle scientific notation
        try:
            if 'e' in latency_str.lower() and 's' not in latency_str:
                return float(latency_str)
        except ValueError:
            pass
        
        # Handle microseconds
        if 'µs' in latency_str or 'us' in latency_str:
            value = float(latency_str.replace('µs', '').replace('us', ''))
            return value / 1_000_000
        
        # Handle milliseconds
        if 'ms' in latency_str:
            value = float(latency_str.replace('ms', ''))
            return value / 1_000
        
        # Handle seconds
        if 's' in latency_str:
            value = float(latency_str.replace('s', ''))
            return value
        
        # Try parsing as raw float (assume seconds)
        try:
            return float(latency_str)
        except ValueError:
            return 0.0

    def parse_stake(self, stake_str: str) -> int:
        """
        Parse stake string to integer.
        Example: "10ulava" -> 10
        """
        # Remove unit suffix
        stake_str = re.sub(r'[a-z]+', '', stake_str, flags=re.IGNORECASE)
        try:
            return int(float(stake_str))
        except ValueError:
            return 0

    def parse_file(self, filepath: str):
        """Parse the CONSUMER.log file with optimized performance."""
        print(f"Parsing {filepath}...", flush=True)
        
        current_ts = None
        line_count = 0
        
        with open(filepath, 'r', buffering=1024*1024) as f:  # 1MB buffer
            for line in f:
                line_count += 1
                if line_count % 100000 == 0:
                    print(f"Processed {line_count:,} lines...", flush=True)
                
                # Extract timestamp - only if line starts with month
                if line and line[0].isupper():
                    ts = self.parse_timestamp(line)
                    if ts:
                        current_ts = ts
                        if self.start_ts is None:
                            self.start_ts = ts
                        self.end_ts = ts
                
                if current_ts is None:
                    continue
                
                # Use simple string checks before regex for speed
                if self.has_choosing in line:
                    # Extract chosenProviders using simple parsing
                    idx = line.find("chosenProviders=")
                    if idx > 0:
                        start = idx + 16
                        end = line.find(" ", start)
                        if end == -1:
                            end = len(line)
                        providers_str = line[start:end].strip()
                        
                        # Handle comma-separated providers
                        for provider in providers_str.split(','):
                            provider = provider.strip()
                            if provider:
                                self.selection_counts[current_ts][provider] += 1
                
                # Parse optimizer probe updates (latency measurements)
                elif self.has_probe in line:
                    # Fast string parsing instead of regex
                    parts = line.split()
                    latency_str = None
                    provider = None
                    success = False
                    
                    for i, part in enumerate(parts):
                        if part.startswith("latency="):
                            latency_str = part[8:]
                        elif part.startswith("providerAddress="):
                            provider = part[16:]
                        elif part.startswith("success="):
                            success = part[8:] == "true"
                    
                    if provider and latency_str:
                        latency = self.parse_latency(latency_str)
                        
                        # Store probe data
                        if provider not in self.providers:
                            self.providers[provider]["name"] = provider
                        
                        # Add as a metric point
                        self.providers[provider]["metrics"].append({
                            "ts": current_ts,
                            "latency": latency,
                            "availability": 1.0 if success else 0.0
                        })
                
                # Parse detailed provider scores
                elif self.has_score in line:
                    # Fast extraction of key fields
                    provider = self._extract_value(line, "provider=")
                    raw_availability = self._extract_float(line, "raw_availability=")
                    raw_latency = self._extract_float(line, "raw_latency_sec=")
                    raw_stake_str = self._extract_value(line, "raw_stake=")
                    raw_sync = self._extract_float(line, "raw_sync_sec=")
                    
                    if provider and raw_stake_str:
                        stake = self.parse_stake(raw_stake_str)
                        
                        # Update provider info
                        if provider not in self.providers:
                            self.providers[provider]["name"] = provider
                        
                        self.providers[provider]["stake"] = stake
                        
                        # Add or update metric
                        if raw_availability is not None and raw_latency is not None and raw_sync is not None:
                            self.providers[provider]["metrics"].append({
                                "ts": current_ts,
                                "availability": raw_availability,
                                "latency": raw_latency,
                                "sync": raw_sync
                            })
        
        print(f"Finished parsing {line_count:,} lines", flush=True)
        print(f"Found {len(self.providers)} providers", flush=True)
        print(f"Time range: {self.start_ts} to {self.end_ts}", flush=True)
    
    def _extract_value(self, line: str, key: str) -> str:
        """Fast extraction of key=value from line."""
        idx = line.find(key)
        if idx < 0:
            return None
        start = idx + len(key)
        end = line.find(" ", start)
        if end == -1:
            end = len(line)
        return line[start:end].strip()
    
    def _extract_float(self, line: str, key: str) -> float:
        """Fast extraction of float value from line."""
        value_str = self._extract_value(line, key)
        if value_str:
            try:
                return float(value_str)
            except ValueError:
                return None
        return None

    def aggregate_metrics(self, sample_interval_ms: int = 60000):
        """
        Aggregate metrics into time buckets.
        
        Args:
            sample_interval_ms: Sampling interval in milliseconds (default: 60000 = 1 minute)
        """
        print("Aggregating metrics...", flush=True)
        sample_interval_sec = sample_interval_ms / 1000
        
        for provider_name, provider_data in self.providers.items():
            metrics = provider_data["metrics"]
            if not metrics:
                continue
            
            # Group metrics by time bucket
            buckets = defaultdict(lambda: {
                "availability": [],
                "latency": [],
                "sync": []
            })
            
            for metric in metrics:
                bucket_ts = int(metric["ts"] / sample_interval_sec) * int(sample_interval_sec)
                
                if "availability" in metric:
                    buckets[bucket_ts]["availability"].append(metric["availability"])
                if "latency" in metric:
                    buckets[bucket_ts]["latency"].append(metric["latency"])
                if "sync" in metric:
                    buckets[bucket_ts]["sync"].append(metric["sync"])
            
            # Calculate averages for each bucket
            aggregated_metrics = []
            for ts in sorted(buckets.keys()):
                bucket = buckets[ts]
                
                avg_availability = sum(bucket["availability"]) / len(bucket["availability"]) if bucket["availability"] else 0
                avg_latency = sum(bucket["latency"]) / len(bucket["latency"]) if bucket["latency"] else 0
                avg_sync = sum(bucket["sync"]) / len(bucket["sync"]) if bucket["sync"] else 0
                
                aggregated_metrics.append({
                    "ts": ts,  # Keep as float to preserve nanosecond precision
                    "availability": round(avg_availability, 4),
                    "latency": round(avg_latency, 6),
                    "sync": round(avg_sync, 4)
                })
            
            provider_data["aggregated_metrics"] = aggregated_metrics
        
        print(f"Aggregation complete", flush=True)

    def build_output(self, sample_interval_ms: int = 60000) -> Dict[str, Any]:
        """
        Build the output JSON structure.
        
        Args:
            sample_interval_ms: Sampling interval in milliseconds
            
        Returns:
            Dictionary matching the target JSON schema
        """
        print("Building output structure...", flush=True)
        
        # Aggregate metrics first
        self.aggregate_metrics(sample_interval_ms)
        
        # Build observed selections
        print("Processing selections...", flush=True)
        observed_selections = []
        for ts in sorted(self.selection_counts.keys()):
            for provider, count in self.selection_counts[ts].items():
                observed_selections.append({
                    "ts": ts,  # Keep as float to preserve nanosecond precision
                    "provider": provider,
                    "count": count
                })
        
        # Build provider series
        print("Building provider series...", flush=True)
        providers_output = []
        for provider_name in sorted(self.providers.keys()):
            provider_data = self.providers[provider_name]
            
            providers_output.append({
                "name": provider_data["name"],
                "stake": provider_data["stake"],
                "series": provider_data.get("aggregated_metrics", [])
            })
        
        # Build final output
        output = {
            "meta": {
                "schemaVersion": "1.0",
                "timeUnit": "seconds",
                "timePrecision": "nanoseconds",
                "latencyUnit": "seconds",
                "syncUnit": "seconds",
                "startTs": self.start_ts or 0,
                "endTs": self.end_ts or 0,
                "sampleIntervalMs": sample_interval_ms
            },
            "providers": providers_output,
            "observedSelections": observed_selections
        }
        
        print("Output structure complete", flush=True)
        return output


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Parse CONSUMER.log and generate structured JSON with provider metrics"
    )
    parser.add_argument(
        "input_file",
        help="Path to CONSUMER.log file"
    )
    parser.add_argument(
        "-o", "--output",
        default="consumer_metrics.json",
        help="Output JSON file (default: consumer_metrics.json)"
    )
    parser.add_argument(
        "-i", "--interval",
        type=int,
        default=60000,
        help="Sampling interval in milliseconds (default: 60000 = 1 minute)"
    )
    parser.add_argument(
        "--pretty",
        action="store_true",
        help="Pretty-print JSON output"
    )
    
    args = parser.parse_args()
    
    # Ensure unbuffered output
    sys.stdout.reconfigure(line_buffering=True)
    
    # Parse the log file
    log_parser = ConsumerLogParser()
    log_parser.parse_file(args.input_file)
    
    # Build output structure
    output = log_parser.build_output(sample_interval_ms=args.interval)
    
    # Write to file
    print(f"Writing output to {args.output}...", flush=True)
    with open(args.output, 'w') as f:
        if args.pretty:
            json.dump(output, f, indent=2)
        else:
            json.dump(output, f)
    
    print(f"\n✅ Output written to {args.output}", flush=True)
    print(f"   Providers: {len(output['providers'])}", flush=True)
    print(f"   Selections: {len(output['observedSelections'])}", flush=True)
    
    # Print summary
    for provider in output['providers']:
        print(f"\n   {provider['name']}:", flush=True)
        print(f"     - Stake: {provider['stake']}", flush=True)
        print(f"     - Data points: {len(provider['series'])}", flush=True)


if __name__ == "__main__":
    main()
