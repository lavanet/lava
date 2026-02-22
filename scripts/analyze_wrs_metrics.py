#!/usr/bin/env python3
"""
WRS Metrics Analyzer

Reads metrics from the provider_optimizer_metrics endpoint and provides
comprehensive analysis of provider scores, parameter contributions, and
selection probabilities.

Usage:
    python analyze_wrs_metrics.py [--url URL] [--chain CHAIN_ID] [--json]
"""

import json
import sys
import argparse
import requests
from typing import Dict, List, Any
from collections import defaultdict


def fetch_metrics(url: str) -> List[Dict[str, Any]]:
    """Fetch metrics from the optimizer endpoint."""
    try:
        response = requests.get(url, timeout=5)
        response.raise_for_status()
        data = response.json()
        if not isinstance(data, list):
            print(f"Error: Expected list, got {type(data)}", file=sys.stderr)
            return []
        return data
    except requests.exceptions.RequestException as e:
        print(f"Error fetching metrics: {e}", file=sys.stderr)
        return []
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON: {e}", file=sys.stderr)
        return []


def verify_score_calculation(provider: Dict[str, Any]) -> Dict[str, Any]:
    """Verify that composite score equals sum of contributions."""
    contrib_sum = (
        provider.get('availability_contribution', 0) +
        provider.get('latency_contribution', 0) +
        provider.get('sync_contribution', 0) +
        provider.get('stake_contribution', 0)
    )
    composite = provider.get('selection_composite', 0)
    diff = abs(composite - contrib_sum)
    
    return {
        'provider': provider.get('provider', 'unknown'),
        'composite_reported': composite,
        'composite_calculated': contrib_sum,
        'difference': diff,
        'valid': diff < 0.0001  # Allow small floating point errors
    }


def calculate_selection_probabilities(providers: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """Calculate expected selection probability for each provider."""
    total_score = sum(p.get('selection_composite', 0) for p in providers)
    
    if total_score == 0:
        return []
    
    results = []
    for p in providers:
        composite = p.get('selection_composite', 0)
        probability = (composite / total_score) * 100.0
        
        results.append({
            'provider': p.get('provider', 'unknown'),
            'composite_score': composite,
            'expected_probability_pct': probability,
            'actual_selection_rate_pct': p.get('selection_rate', 0) * 100.0,
            'selection_count': p.get('selection_count', 0),
        })
    
    return results


def analyze_parameter_impact(providers: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Analyze which parameters have the most impact on selection."""
    if not providers:
        return {}
    
    # Calculate average contributions
    total_providers = len(providers)
    avg_contributions = {
        'availability': sum(p.get('availability_contribution', 0) for p in providers) / total_providers,
        'latency': sum(p.get('latency_contribution', 0) for p in providers) / total_providers,
        'sync': sum(p.get('sync_contribution', 0) for p in providers) / total_providers,
        'stake': sum(p.get('stake_contribution', 0) for p in providers) / total_providers,
    }
    
    # Calculate variance (spread) in contributions
    variance = {}
    for param in ['availability', 'latency', 'sync', 'stake']:
        key = f'{param}_contribution'
        values = [p.get(key, 0) for p in providers]
        mean = avg_contributions[param]
        variance[param] = sum((v - mean) ** 2 for v in values) / total_providers
    
    return {
        'average_contributions': avg_contributions,
        'variance': variance,
        'total_avg': sum(avg_contributions.values()),
    }


def print_summary(providers: List[Dict[str, Any]], chain_id: str = None):
    """Print comprehensive summary of metrics."""
    if not providers:
        print("No providers found in metrics")
        return
    
    print("=" * 80)
    print(f"WRS METRICS ANALYSIS{f' - Chain: {chain_id}' if chain_id else ''}")
    print("=" * 80)
    print()
    
    # 1. Provider Count
    print(f"📊 Total Providers: {len(providers)}")
    print()
    
    # 2. Score Verification
    print("✅ SCORE VERIFICATION")
    print("-" * 80)
    all_valid = True
    for p in providers:
        verification = verify_score_calculation(p)
        status = "✓" if verification['valid'] else "✗"
        print(f"{status} {verification['provider']:<40} "
              f"Composite: {verification['composite_reported']:.4f} "
              f"(calc: {verification['composite_calculated']:.4f}, "
              f"diff: {verification['difference']:.6f})")
        if not verification['valid']:
            all_valid = False
    
    if all_valid:
        print("\n✅ All score calculations are correct!")
    else:
        print("\n⚠️  Some scores have discrepancies!")
    print()
    
    # 3. Selection Probabilities
    print("🎯 SELECTION PROBABILITIES")
    print("-" * 80)
    probabilities = calculate_selection_probabilities(providers)
    print(f"{'Provider':<40} {'Score':<10} {'Expected':<12} {'Actual':<12} {'Count':<10}")
    print("-" * 80)
    for p in sorted(probabilities, key=lambda x: x['expected_probability_pct'], reverse=True):
        print(f"{p['provider']:<40} "
              f"{p['composite_score']:<10.4f} "
              f"{p['expected_probability_pct']:<12.2f}% "
              f"{p['actual_selection_rate_pct']:<12.2f}% "
              f"{p['selection_count']:<10}")
    print()
    
    # 4. Parameter Contributions
    print("📈 PARAMETER CONTRIBUTIONS")
    print("-" * 80)
    print(f"{'Provider':<40} {'Avail':<10} {'Latency':<10} {'Sync':<10} {'Stake':<10} {'Total':<10}")
    print("-" * 80)
    for p in providers:
        avail = p.get('availability_contribution', 0)
        latency = p.get('latency_contribution', 0)
        sync = p.get('sync_contribution', 0)
        stake = p.get('stake_contribution', 0)
        total = avail + latency + sync + stake
        print(f"{p.get('provider', 'unknown'):<40} "
              f"{avail:<10.4f} "
              f"{latency:<10.4f} "
              f"{sync:<10.4f} "
              f"{stake:<10.4f} "
              f"{total:<10.4f}")
    print()
    
    # 5. Normalized Scores
    print("🔢 NORMALIZED SCORES (0-1 scale, higher is better)")
    print("-" * 80)
    print(f"{'Provider':<40} {'Avail':<10} {'Latency':<10} {'Sync':<10} {'Stake':<10}")
    print("-" * 80)
    for p in providers:
        print(f"{p.get('provider', 'unknown'):<40} "
              f"{p.get('selection_availability', 0):<10.4f} "
              f"{p.get('selection_latency', 0):<10.4f} "
              f"{p.get('selection_sync', 0):<10.4f} "
              f"{p.get('selection_stake', 0):<10.4f}")
    print()
    
    # 6. Raw EWMA Values
    print("📊 RAW EWMA VALUES (before normalization)")
    print("-" * 80)
    print(f"{'Provider':<40} {'Avail':<10} {'Latency(s)':<12} {'Sync(s)':<12} {'Stake':<12}")
    print("-" * 80)
    for p in providers:
        print(f"{p.get('provider', 'unknown'):<40} "
              f"{p.get('availability_score', 0):<10.4f} "
              f"{p.get('latency_score', 0):<12.6f} "
              f"{p.get('sync_score', 0):<12.6f} "
              f"{p.get('provider_stake', 0):<12}")
    print()
    
    # 7. Parameter Impact Analysis
    print("💡 PARAMETER IMPACT ANALYSIS")
    print("-" * 80)
    impact = analyze_parameter_impact(providers)
    print(f"Average Contributions:")
    for param, value in impact['average_contributions'].items():
        percentage = (value / impact['total_avg'] * 100) if impact['total_avg'] > 0 else 0
        print(f"  {param.capitalize():<15} {value:.4f}  ({percentage:.1f}% of total)")
    print()
    print(f"Variance (spread) in Contributions:")
    for param, value in impact['variance'].items():
        print(f"  {param.capitalize():<15} {value:.6f}")
    print()
    print("💡 Higher variance = parameter differentiates providers more")
    print()


def print_json_output(providers: List[Dict[str, Any]]):
    """Print metrics in JSON format."""
    output = {
        'providers': providers,
        'summary': {
            'total_providers': len(providers),
            'probabilities': calculate_selection_probabilities(providers),
            'parameter_impact': analyze_parameter_impact(providers),
            'verification': [verify_score_calculation(p) for p in providers],
        }
    }
    print(json.dumps(output, indent=2))


def main():
    parser = argparse.ArgumentParser(description='Analyze WRS provider optimizer metrics')
    parser.add_argument('--url', default='http://localhost:7779/provider_optimizer_metrics',
                        help='Metrics endpoint URL (default: http://localhost:7779/provider_optimizer_metrics)')
    parser.add_argument('--chain', help='Filter by chain ID')
    parser.add_argument('--json', action='store_true', help='Output as JSON')
    
    args = parser.parse_args()
    
    # Fetch metrics
    providers = fetch_metrics(args.url)
    
    if not providers:
        print("No metrics data available", file=sys.stderr)
        return 1
    
    # Filter by chain if requested
    if args.chain:
        providers = [p for p in providers if p.get('chain_id') == args.chain]
        if not providers:
            print(f"No providers found for chain: {args.chain}", file=sys.stderr)
            return 1
    
    # Output
    if args.json:
        print_json_output(providers)
    else:
        print_summary(providers, args.chain)
    
    return 0


if __name__ == '__main__':
    sys.exit(main())
