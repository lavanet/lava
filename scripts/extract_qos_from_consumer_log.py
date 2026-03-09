#!/usr/bin/env python3
"""
Extract QoS values from consumer log for each request.

This script parses the consumer.log file and extracts non-normalized QoS values
(latency, availability, sync, stake) for each provider and identifies which
provider was selected for each request.
"""

import re
import json
import sys
from collections import defaultdict
from typing import Dict, List, Any


def parse_qos_excellence(line: str) -> Dict[str, Any]:
    """
    Parse a QoS Excellence log line to extract provider and QoS metrics.
    
    Example:
    TRC [Optimizer] QoS Excellence for provider address=primary-2220-jsonrpc 
    report="latency:\"937080000000000\" availability:\"1000000000000000000\" sync:\"100000000000000000\" "
    """
    result = {}
    
    # Extract provider address
    provider_match = re.search(r'address=([^\s]+)', line)
    if provider_match:
        result['provider'] = provider_match.group(1)
    
    # Extract report content
    report_match = re.search(r'report="([^"]*)"', line)
    if report_match:
        report = report_match.group(1)
        
        # Extract latency
        latency_match = re.search(r'latency:\\"([^\\"]+)\\"', report)
        if latency_match:
            result['latency'] = latency_match.group(1)
        
        # Extract availability
        availability_match = re.search(r'availability:\\"([^\\"]+)\\"', report)
        if availability_match:
            result['availability'] = availability_match.group(1)
        
        # Extract sync
        sync_match = re.search(r'sync:\\"([^\\"]+)\\"', report)
        if sync_match:
            result['sync'] = sync_match.group(1)
    
    return result


def parse_provider_score_breakdown(line: str) -> Dict[str, Any]:
    """
    Parse a provider score calculation breakdown to extract raw QoS values.
    
    Example:
    DBG Provider score calculation breakdown ... provider=primary-2222-jsonrpc 
    raw_availability=1 raw_latency_sec=1e-08 raw_stake=10ulava raw_sync_sec=0.1 ...
    """
    result = {}
    
    # Extract provider
    provider_match = re.search(r'provider=([^\s]+)', line)
    if provider_match:
        result['provider'] = provider_match.group(1)
    
    # Extract raw latency
    latency_match = re.search(r'raw_latency_sec=([^\s]+)', line)
    if latency_match:
        result['latency'] = latency_match.group(1)
    
    # Extract raw availability
    availability_match = re.search(r'raw_availability=([^\s]+)', line)
    if availability_match:
        result['availability'] = availability_match.group(1)
    
    # Extract raw sync
    sync_match = re.search(r'raw_sync_sec=([^\s]+)', line)
    if sync_match:
        result['sync'] = sync_match.group(1)
    
    # Extract raw stake
    stake_match = re.search(r'raw_stake=([^\s]+)', line)
    if stake_match:
        stake_str = stake_match.group(1)
        # Remove "ulava" suffix if present
        stake_str = stake_str.replace('ulava', '')
        result['stake'] = stake_str
    
    return result


def parse_provider_selection(line: str) -> Dict[str, Any]:
    """
    Parse a provider selection completed line to extract selected provider.
    
    Example:
    DBG Provider selection completed ... selected_provider=primary-2222-jsonrpc ...
    """
    result = {}
    
    # Extract selected provider
    provider_match = re.search(r'selected_provider=([^\s]+)', line)
    if provider_match:
        result['selected_provider'] = provider_match.group(1)
    
    return result


def extract_guid(line: str) -> str:
    """Extract GUID from log line if present."""
    guid_match = re.search(r'GUID=(\d+)', line)
    if guid_match:
        return guid_match.group(1)
    return None


def process_log_file(log_path: str) -> List[Dict[str, Any]]:
    """
    Process the consumer log file and extract QoS data for each request.
    
    Returns a list of request objects, each containing:
    - request_id: Sequential request number
    - timestamp: The timestamp of the selection
    - providers: Dict of provider data with QoS metrics
    - selected_provider: The provider that was selected
    """
    requests = []
    current_batch_providers = {}  # Temporary storage for providers before selection
    request_counter = 0
    
    with open(log_path, 'r', encoding='utf-8', errors='ignore') as f:
        for line in f:
            # Parse provider score breakdown lines for raw QoS values
            if 'Provider score calculation breakdown' in line:
                score_data = parse_provider_score_breakdown(line)
                if score_data and 'provider' in score_data:
                    provider = score_data['provider']
                    current_batch_providers[provider] = score_data
            
            # Parse provider selection lines - this marks the end of a request
            elif 'Provider selection completed' in line:
                selection_data = parse_provider_selection(line)
                if selection_data and 'selected_provider' in selection_data:
                    # Extract timestamp
                    timestamp_match = re.match(r'^(\w+\s+\d+\s+\d+:\d+:\d+)', line)
                    timestamp = timestamp_match.group(1) if timestamp_match else None
                    
                    # Extract GUID if present
                    guid = extract_guid(line)
                    
                    # Create request object
                    request_counter += 1
                    request = {
                        'request_id': request_counter,
                        'timestamp': timestamp,
                        'guid': guid,
                        'providers': dict(current_batch_providers),  # Copy the dict
                        'selected_provider': selection_data['selected_provider']
                    }
                    
                    requests.append(request)
                    
                    # Clear the batch for next request
                    current_batch_providers = {}
    
    return requests


def print_summary(requests: List[Dict[str, Any]]):
    """Print a summary of the extracted data."""
    from collections import Counter
    
    if not requests:
        print("No requests found.")
        return
    
    print(f"\n{'='*60}")
    print("QoS EXTRACTION SUMMARY")
    print(f"{'='*60}")
    
    print(f"\nTotal requests: {len(requests)}")
    print(f"Timestamp range: {requests[0]['timestamp']} to {requests[-1]['timestamp']}")
    
    # Count selected providers
    selected = Counter(req['selected_provider'] for req in requests)
    print('\nSelected provider distribution:')
    for provider, count in selected.most_common():
        print(f'  {provider:30s}: {count:6d} ({count/len(requests)*100:5.1f}%)')
    
    # Count how many providers per request
    provider_counts = Counter(len(req['providers']) for req in requests)
    print('\nProviders evaluated per request:')
    for count in sorted(provider_counts.keys()):
        freq = provider_counts[count]
        print(f'  {count} provider{"s" if count != 1 else " "}: {freq:6d} requests ({freq/len(requests)*100:5.1f}%)')
    
    # Find requests with varying QoS
    varying_latency = 0
    varying_sync = 0
    varying_availability = 0
    
    for req in requests:
        if len(req['providers']) > 1:
            latencies = set(p['latency'] for p in req['providers'].values())
            syncs = set(p['sync'] for p in req['providers'].values())
            availabilities = set(p['availability'] for p in req['providers'].values())
            
            if len(latencies) > 1:
                varying_latency += 1
            if len(syncs) > 1:
                varying_sync += 1
            if len(availabilities) > 1:
                varying_availability += 1
    
    total_multi = sum(1 for req in requests if len(req['providers']) > 1)
    if total_multi > 0:
        print('\nQoS variance across providers (in multi-provider requests):')
        print(f'  Varying latency:      {varying_latency:6d} ({varying_latency/total_multi*100:5.1f}%)')
        print(f'  Varying sync:         {varying_sync:6d} ({varying_sync/total_multi*100:5.1f}%)')
        print(f'  Varying availability: {varying_availability:6d} ({varying_availability/total_multi*100:5.1f}%)')
    
    print(f"\n{'='*60}\n")


def main():
    if len(sys.argv) < 2:
        print("Usage: python extract_qos_from_consumer_log.py <path_to_consumer.log> [output.json] [--summary]")
        print("\nExample:")
        print("  python extract_qos_from_consumer_log.py testutil/debugging/logs/CONSUMER.log")
        print("  python extract_qos_from_consumer_log.py testutil/debugging/logs/CONSUMER.log output.json")
        print("  python extract_qos_from_consumer_log.py testutil/debugging/logs/CONSUMER.log output.json --summary")
        print("\nOptions:")
        print("  --summary    Show summary statistics (always shown when output file is specified)")
        sys.exit(1)
    
    log_path = sys.argv[1]
    output_path = None
    show_summary = '--summary' in sys.argv
    
    # Check for output path (must not be a flag)
    if len(sys.argv) > 2 and not sys.argv[2].startswith('--'):
        output_path = sys.argv[2]
        show_summary = True  # Always show summary when writing to file
    
    print(f"Processing log file: {log_path}")
    
    try:
        requests = process_log_file(log_path)
        
        print(f"Extracted {len(requests)} requests with complete QoS data")
        
        # Show summary if requested or if writing to file
        if show_summary:
            print_summary(requests)
        
        # Output as JSON
        output_json = json.dumps(requests, indent=2)
        
        if output_path:
            with open(output_path, 'w') as f:
                f.write(output_json)
            print(f"Output written to: {output_path}")
        else:
            print("\nJSON Output:")
            print(output_json)
    
    except FileNotFoundError:
        print(f"Error: File not found: {log_path}")
        sys.exit(1)
    except Exception as e:
        print(f"Error processing log file: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
