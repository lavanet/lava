#!/usr/bin/env python3
"""
Example script to analyze QoS data extracted from consumer logs.

This demonstrates how to work with the JSON output from extract_qos_from_consumer_log.py.
"""

import json
import sys
from collections import defaultdict, Counter
from typing import Dict, List


def load_qos_data(filepath: str) -> List[Dict]:
    """Load QoS data from JSON file."""
    with open(filepath, 'r') as f:
        return json.load(f)


def analyze_provider_selection(data: List[Dict]):
    """Analyze which providers were selected and why."""
    print("\n" + "="*80)
    print("PROVIDER SELECTION ANALYSIS")
    print("="*80)
    
    selected_counts = Counter(req['selected_provider'] for req in data)
    
    print(f"\nTotal requests: {len(data)}")
    print("\nProvider selection frequency:")
    for provider, count in selected_counts.most_common():
        pct = (count / len(data)) * 100
        print(f"  {provider:30s}: {count:6d} ({pct:5.1f}%)")


def analyze_qos_metrics(data: List[Dict]):
    """Analyze QoS metrics for each provider."""
    print("\n" + "="*80)
    print("QoS METRICS ANALYSIS")
    print("="*80)
    
    # Aggregate metrics by provider
    provider_metrics = defaultdict(lambda: {
        'latencies': [],
        'availabilities': [],
        'syncs': [],
        'stakes': [],
        'times_considered': 0,
        'times_selected': 0
    })
    
    for req in data:
        selected = req['selected_provider']
        
        for provider_name, qos in req['providers'].items():
            metrics = provider_metrics[provider_name]
            metrics['times_considered'] += 1
            
            if provider_name == selected:
                metrics['times_selected'] += 1
            
            # Convert string values to floats for analysis
            try:
                metrics['latencies'].append(float(qos['latency']))
                metrics['availabilities'].append(float(qos['availability']))
                metrics['syncs'].append(float(qos['sync']))
                metrics['stakes'].append(float(qos['stake']))
            except (ValueError, KeyError):
                pass
    
    # Calculate and display statistics
    for provider_name, metrics in sorted(provider_metrics.items()):
        print(f"\n{provider_name}:")
        print(f"  Considered in {metrics['times_considered']} requests")
        print(f"  Selected {metrics['times_selected']} times ({metrics['times_selected']/metrics['times_considered']*100:.1f}%)")
        
        if metrics['latencies']:
            avg_latency = sum(metrics['latencies']) / len(metrics['latencies'])
            min_latency = min(metrics['latencies'])
            max_latency = max(metrics['latencies'])
            print(f"  Latency: avg={avg_latency:.6f}s, min={min_latency:.6f}s, max={max_latency:.6f}s")
        
        if metrics['availabilities']:
            avg_avail = sum(metrics['availabilities']) / len(metrics['availabilities'])
            print(f"  Availability: avg={avg_avail:.6f}")
        
        if metrics['syncs']:
            avg_sync = sum(metrics['syncs']) / len(metrics['syncs'])
            min_sync = min(metrics['syncs'])
            max_sync = max(metrics['syncs'])
            print(f"  Sync: avg={avg_sync:.6f}s, min={min_sync:.6f}s, max={max_sync:.6f}s")
        
        if metrics['stakes']:
            avg_stake = sum(metrics['stakes']) / len(metrics['stakes'])
            print(f"  Stake: avg={avg_stake:.2f}")


def analyze_competitive_selections(data: List[Dict]):
    """Analyze requests where multiple providers competed."""
    print("\n" + "="*80)
    print("COMPETITIVE SELECTION ANALYSIS")
    print("="*80)
    
    # Group by number of providers
    by_provider_count = defaultdict(list)
    for req in data:
        count = len(req['providers'])
        by_provider_count[count].append(req)
    
    print(f"\nRequests by number of competing providers:")
    for count in sorted(by_provider_count.keys()):
        reqs = by_provider_count[count]
        print(f"  {count} provider{'s' if count != 1 else ' '}: {len(reqs):6d} requests ({len(reqs)/len(data)*100:5.1f}%)")
    
    # Analyze 3-way competitions
    if 3 in by_provider_count:
        three_way = by_provider_count[3]
        print(f"\n3-way competitions: {len(three_way)} requests")
        
        # Count who wins in 3-way
        winners = Counter(req['selected_provider'] for req in three_way)
        print("  Winner distribution:")
        for provider, count in winners.most_common():
            print(f"    {provider:30s}: {count:6d} ({count/len(three_way)*100:5.1f}%)")


def find_interesting_cases(data: List[Dict], limit: int = 5):
    """Find interesting edge cases."""
    print("\n" + "="*80)
    print("INTERESTING CASES")
    print("="*80)
    
    # Find cases where lower latency didn't win
    print(f"\nCases where selected provider didn't have lowest latency (showing up to {limit}):")
    
    count = 0
    for req in data:
        if len(req['providers']) < 2:
            continue
        
        # Get latencies
        latencies = {}
        for p_name, qos in req['providers'].items():
            try:
                latencies[p_name] = float(qos['latency'])
            except (ValueError, KeyError):
                pass
        
        if not latencies:
            continue
        
        lowest_latency_provider = min(latencies, key=latencies.get)
        selected = req['selected_provider']
        
        if lowest_latency_provider != selected:
            print(f"\n  Request {req['request_id']}:")
            print(f"    Selected: {selected} (latency: {latencies.get(selected, 'N/A')}s)")
            print(f"    Lowest latency: {lowest_latency_provider} (latency: {latencies[lowest_latency_provider]}s)")
            print(f"    All providers:")
            for p_name, qos in req['providers'].items():
                marker = " <- SELECTED" if p_name == selected else ""
                print(f"      {p_name}: latency={qos.get('latency', 'N/A')}s, "
                      f"sync={qos.get('sync', 'N/A')}s, "
                      f"availability={qos.get('availability', 'N/A')}{marker}")
            
            count += 1
            if count >= limit:
                break
    
    if count == 0:
        print("  None found (selected provider always had lowest latency)")


def main():
    if len(sys.argv) < 2:
        print("Usage: python analyze_qos_output.py <qos_output.json>")
        print("\nExample:")
        print("  python analyze_qos_output.py qos_output.json")
        sys.exit(1)
    
    filepath = sys.argv[1]
    
    try:
        print(f"Loading QoS data from: {filepath}")
        data = load_qos_data(filepath)
        print(f"Loaded {len(data)} requests")
        
        # Run analyses
        analyze_provider_selection(data)
        analyze_qos_metrics(data)
        analyze_competitive_selections(data)
        find_interesting_cases(data)
        
        print("\n" + "="*80)
        print("Analysis complete!")
        print("="*80 + "\n")
        
    except FileNotFoundError:
        print(f"Error: File not found: {filepath}")
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"Error: Invalid JSON file: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
