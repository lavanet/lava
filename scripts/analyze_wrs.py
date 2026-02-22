import sys
import re
from collections import defaultdict
import argparse
import os
import math

# Set backend to Agg before importing pyplot to avoid GUI dependencies
try:
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
    import matplotlib.gridspec as gridspec
    HAS_MATPLOTLIB = True
except ImportError as e:
    print(f"Matplotlib import error: {e}")
    HAS_MATPLOTLIB = False

# Default weights from WeightedSelectorConfig
DEFAULT_WEIGHTS = {
    'availability': 0.3,
    'latency': 0.3,
    'sync': 0.2,
    'stake': 0.2
}

def parse_log_line(line, json_format=False):
    if "Provider selection completed" not in line:
        return None
    
    # JSON format parsing
    if json_format:
        import json
        try:
            # Check if the line contains JSON (look for opening brace after timestamp)
            json_start = line.find('{')
            if json_start >= 0:
                json_str = line[json_start:]
                data = json.loads(json_str)
                
                # Check if this is a provider selection log
                if data.get('message') == 'Provider selection completed':
                    # Convert JSON keys to the format expected by the rest of the script
                    params = {}
                    for key, value in data.items():
                        if key != 'message' and key != 'severity' and key != 'time':
                            params[key] = str(value)
                    
                    # Extract timestamp from the beginning of the line (if present)
                    timestamp = line[:json_start].strip() or line[:15]
                    params['timestamp'] = timestamp
                    
                    return params
        except (json.JSONDecodeError, ValueError):
            return None
    
    # Text-based parsing (default format)
    # Extract timestamp if possible (e.g. "Feb  2 14:55:13")
    timestamp = line[:15]
    
    try:
        _, params_str = line.split("Provider selection completed", 1)
    except ValueError:
        return None
    
    params = {}
    parts = params_str.strip().split()
    for part in parts:
        if '=' in part:
            key, value = part.split('=', 1)
            params[key] = value
    
    # Extract GUID from the context attribute if present
    # The GUID can be in several formats:
    # 1. GUID={context...guid:1234567890...}
    # 2. GUID=1234567890
    # 3. GUID={...}1234567890
    if 'GUID' in params:
        guid_str = params['GUID']
        # Try different patterns to extract the GUID
        
        # Pattern 1: guid:NUMBER format (inside context structure)
        guid_match = re.search(r'guid:(\d+)', guid_str)
        if guid_match:
            params['GUID'] = guid_match.group(1)
        # Pattern 2: Just a number
        elif guid_str.isdigit():
            params['GUID'] = guid_str
        # Pattern 3: Extract any number from the string
        else:
            number_match = re.search(r'(\d+)', guid_str)
            if number_match:
                params['GUID'] = number_match.group(1)
            else:
                # If we can't parse it, use the whole string
                params['GUID'] = guid_str
            
    params['timestamp'] = timestamp
    return params

def verify_calculations(data, weights):
    """
    Verify that composite scores and probabilities match the expected formulas.
    Returns a list of verification errors/warnings.
    """
    warnings = []
    num_candidates = int(data.get('num_candidates', 0))
    total_score_log = float(data.get('total_score', 0))
    
    calculated_total_score = 0.0
    
    for i in range(1, num_candidates + 1):
        prefix = f"candidate_{i}"
        provider = data.get(f"{prefix}_provider")
        if not provider:
            continue
            
        # Get normalized scores
        avail = float(data.get(f"{prefix}_availability", 0))
        latency = float(data.get(f"{prefix}_latency", 0))
        sync = float(data.get(f"{prefix}_sync", 0))
        stake = float(data.get(f"{prefix}_stake", 0))
        composite_log = float(data.get(f"{prefix}_composite", 0))
        score_log = float(data.get(f"{prefix}_score", 0))
        
        # Verify Composite Score
        # composite = avail*w_a + lat*w_l + sync*w_s + stake*w_st
        calc_composite = (avail * weights['availability'] + 
                          latency * weights['latency'] + 
                          sync * weights['sync'] + 
                          stake * weights['stake'])
        
        # Allow small floating point error
        if abs(calc_composite - composite_log) > 0.001:
            warnings.append(f"Composite mismatch for {provider}: Log={composite_log:.4f}, Calc={calc_composite:.4f}")
            
        calculated_total_score += score_log

    # Verify Total Score
    if abs(calculated_total_score - total_score_log) > 0.001:
        warnings.append(f"Total Score mismatch: Log={total_score_log:.4f}, Calc={calculated_total_score:.4f}")

    # Verify Probabilities
    for i in range(1, num_candidates + 1):
        prefix = f"candidate_{i}"
        score_log = float(data.get(f"{prefix}_score", 0))
        prob_log = float(data.get(f"{prefix}_probability_pct", 0))
        
        if total_score_log > 0:
            calc_prob = (score_log / total_score_log) * 100.0
            if abs(calc_prob - prob_log) > 0.01:
                 warnings.append(f"Probability mismatch for {provider}: Log={prob_log:.2f}%, Calc={calc_prob:.2f}%")

    return warnings

def analyze_logs(log_file, output_graph=None, skip_initial_events=500, json_format=False):
    if not os.path.exists(log_file):
        print(f"Error: File {log_file} not found.")
        return

    stats = {
        'total_selections': 0,
        'providers': defaultdict(lambda: {
            'selections': 0,
            'total_probability': 0.0,
            'count_probability': 0,
            'history_prob': [],
            'history_score': [],
            'history_composite': [],
            'history_latency': [],
            'history_sync': [],
            'history_avail': [],
            'history_stake': [],
            'last_score': 0.0,
            'last_latency': 0.0,
            'last_sync': 0.0,
            'last_availability': 0.0,
            'last_stake': 0.0,
            'last_composite': 0.0
        }),
        'rng_values': [],
        'normalized_rng_values': [],
        'total_scores': [],
        'verification_errors': 0,
        'by_guid': defaultdict(list)  # Group selections by GUID
    }

    print(f"Analyzing {log_file}...")
    
    with open(log_file, 'r') as f:
        for line_idx, line in enumerate(f):
            data = parse_log_line(line, json_format=json_format)
            if not data:
                continue
            
            stats['total_selections'] += 1
            selected_provider = data.get('selected_provider')
            
            # Track by GUID if available
            guid = data.get('GUID')
            if guid:
                stats['by_guid'][guid].append({
                    'line': line_idx + 1,
                    'selected_provider': selected_provider,
                    'timestamp': data.get('timestamp'),
                    'total_score': float(data.get('total_score', 0)),
                    'random_value': float(data.get('random_value', 0)),
                    'num_candidates': int(data.get('num_candidates', 0))
                })
            
            if selected_provider:
                stats['providers'][selected_provider]['selections'] += 1

            # Capture RNG value and Total Score
            rng_val = data.get('random_value')
            total_score = data.get('total_score')
            
            if rng_val and total_score:
                try:
                    r = float(rng_val)
                    t = float(total_score)
                    if t > 0:
                        stats['rng_values'].append(r)
                        stats['total_scores'].append(t)
                        stats['normalized_rng_values'].append(r / t)
                except ValueError:
                    pass

            # Verify calculations
            errors = verify_calculations(data, DEFAULT_WEIGHTS)
            if errors:
                stats['verification_errors'] += len(errors)
                if stats['verification_errors'] <= 5: # Only print first few errors
                    print(f"Line {line_idx+1} Warning: {errors[0]}")

            # Process candidates
            num_candidates = int(data.get('num_candidates', 0))
            for i in range(1, num_candidates + 1):
                prefix = f"candidate_{i}"
                provider = data.get(f"{prefix}_provider")
                if provider:
                    prob = float(data.get(f"{prefix}_probability_pct", 0))
                    score_val = float(data.get(f"{prefix}_score", 0))
                    
                    p_stats = stats['providers'][provider]
                    p_stats['total_probability'] += prob
                    p_stats['count_probability'] += 1
                    
                    # Store history
                    p_stats['history_prob'].append(prob)
                    p_stats['history_score'].append(score_val)
                    
                    # Update metrics
                    lat = float(data.get(f"{prefix}_latency", 0))
                    sync = float(data.get(f"{prefix}_sync", 0))
                    avail = float(data.get(f"{prefix}_availability", 0))
                    stake = float(data.get(f"{prefix}_stake", 0))
                    comp = float(data.get(f"{prefix}_composite", 0))
                    
                    p_stats['history_composite'].append(comp)
                    p_stats['history_latency'].append(lat)
                    p_stats['history_sync'].append(sync)
                    p_stats['history_avail'].append(avail)
                    p_stats['history_stake'].append(stake)

                    p_stats['last_score'] = score_val
                    p_stats['last_latency'] = lat
                    p_stats['last_sync'] = sync
                    p_stats['last_availability'] = avail
                    p_stats['last_stake'] = stake
                    p_stats['last_composite'] = comp

    # --- Text Output ---
    print(f"\nAnalysis Results")
    print(f"Total Selections: {stats['total_selections']}")
    
    # GUID Statistics
    num_guids = len(stats['by_guid'])
    if num_guids > 0:
        print(f"Unique Requests (GUIDs): {num_guids}")
        avg_selections_per_guid = stats['total_selections'] / num_guids if num_guids > 0 else 0
        print(f"Avg Selections per Request: {avg_selections_per_guid:.2f}")
        
        # Calculate min/max selections per request
        selections_per_guid = [len(selections) for selections in stats['by_guid'].values()]
        min_selections = min(selections_per_guid)
        max_selections = max(selections_per_guid)
        print(f"Min/Max Selections per Request: {min_selections}/{max_selections}")
        
        # Count requests with multiple attempts (retries)
        multi_attempt_requests = sum(1 for count in selections_per_guid if count > 1)
        if multi_attempt_requests > 0:
            retry_pct = (multi_attempt_requests / num_guids) * 100
            print(f"Requests with Multiple Attempts: {multi_attempt_requests} ({retry_pct:.1f}%)")
    else:
        print(f"GUID tracking: Not available (logs may not contain GUID field)")
    
    # Verification Status
    if stats['verification_errors'] > 0:
         print(f"Verification: WARNING - {stats['verification_errors']} mismatches found (Calculated values didn't match logs)")
    else:
         print(f"Verification: OK (All calculations match formulas)")

    # Distribution Health Check
    max_diff = 0.0
    for p_stats in stats['providers'].values():
        if stats['total_selections'] > 0:
            sel_pct = (p_stats['selections'] / stats['total_selections'] * 100)
            # Correct Expected % calculation:
            avg_prob = (p_stats['total_probability'] / stats['total_selections']) if stats['total_selections'] > 0 else 0
            diff = abs(sel_pct - avg_prob)
            if diff > max_diff:
                max_diff = diff
    
    health = "Good"
    if max_diff > 5.0:
        health = "Poor (>5% deviation)"
    elif max_diff > 2.0:
        health = "Fair (>2% deviation)"
        
    print(f"Distribution Health: {health} (Max Deviation: {max_diff:.2f}%)")
         
    # --- RNG Statistics ---
    rng_vals = stats['rng_values']
    norm_rng_vals = stats['normalized_rng_values']
    total_scores = stats['total_scores']
    
    if rng_vals:
        n = len(rng_vals)
        
        # Raw RNG Stats
        avg_rng = sum(rng_vals) / n
        min_rng = min(rng_vals)
        max_rng = max(rng_vals)
        
        # Normalized RNG Stats (Should be ~0.5 mean, uniform [0,1])
        avg_norm = sum(norm_rng_vals) / n
        min_norm = min(norm_rng_vals)
        max_norm = max(norm_rng_vals)
        
        # Total Score Stats
        avg_score = sum(total_scores) / n
        min_score = min(total_scores)
        max_score = max(total_scores)

        print(f"\nRNG Statistics (Raw Value in [0, TotalScore]):")
        print(f"  Count: {n}")
        print(f"  Mean:   {avg_rng:.4f}")
        print(f"  Min:    {min_rng:.4f}")
        print(f"  Max:    {max_rng:.4f}")
        
        print(f"\nTotal Score Statistics (Upper Bound for RNG):")
        print(f"  Mean:   {avg_score:.4f}")
        print(f"  Min:    {min_score:.4f}")
        print(f"  Max:    {max_score:.4f}")
        
        print(f"\nNormalized RNG Statistics (Value / TotalScore, Expected Uniform [0,1]):")
        print(f"  Mean:   {avg_norm:.4f} (Expected ~0.5000)")
        print(f"  Min:    {min_norm:.4f}")
        print(f"  Max:    {max_norm:.4f}")
        
        # Simple ASCII Histogram for Normalized RNG
        print("\nNormalized RNG Distribution (Text) - Should be flat:")
        num_buckets = 10
        bucket_size = 0.1
        buckets = [0] * num_buckets
        for val in norm_rng_vals:
            idx = int(val / bucket_size)
            if idx >= num_buckets: idx = num_buckets - 1
            buckets[idx] += 1
            
        max_count = max(buckets)
        for i in range(num_buckets):
            low = i * bucket_size
            high = (i + 1) * bucket_size
            count = buckets[i]
            bar_len = int((count / max_count) * 50) if max_count > 0 else 0
            bar = '#' * bar_len
            print(f"  {low:3.1f} - {high:3.1f}: {count:4d} |{bar}")
    else:
        print("RNG Statistics: No data found")

    print("=" * 160)
    print(f"{'Provider':<25} | {'Actual %':<8} | {'Exp %':<8} | {'Diff %':<8} | {'Avg Score':<9} | {'Avg Comp':<9} | {'Avg Lat':<8} | {'Avg Sync':<8} | {'Avg Avail':<9} | {'Avg Stake':<9}")
    print("-" * 160)

    sorted_providers = sorted(stats['providers'].items())
    
    providers_list = []
    actual_pcts = []
    exp_pcts = []

    for provider, p_stats in sorted_providers:
        selection_pct = (p_stats['selections'] / stats['total_selections'] * 100) if stats['total_selections'] > 0 else 0
        avg_prob = (p_stats['total_probability'] / stats['total_selections']) if stats['total_selections'] > 0 else 0
        diff = selection_pct - avg_prob
        
        providers_list.append(provider)
        actual_pcts.append(selection_pct)
        exp_pcts.append(avg_prob)
        
        # Calculate Averages
        count = len(p_stats['history_score'])
        avg_score = sum(p_stats['history_score']) / count if count > 0 else 0
        avg_comp = sum(p_stats['history_composite']) / count if count > 0 else 0
        avg_lat = sum(p_stats['history_latency']) / count if count > 0 else 0
        avg_sync = sum(p_stats['history_sync']) / count if count > 0 else 0
        avg_avail = sum(p_stats['history_avail']) / count if count > 0 else 0
        avg_stake = sum(p_stats['history_stake']) / count if count > 0 else 0

        print(f"{provider:<25} | {selection_pct:6.2f}% | {avg_prob:6.2f}% | {diff:6.2f}% | {avg_score:9.4f} | {avg_comp:9.4f} | {avg_lat:8.4f} | {avg_sync:8.4f} | {avg_avail:9.4f} | {avg_stake:9.4f}")
    print("=" * 160)
    
    # --- Parameter Breakdown Table ---
    print(f"\nParameter Breakdown (Contribution to Score)")
    print(f"Weights: Avail={DEFAULT_WEIGHTS['availability']}, Latency={DEFAULT_WEIGHTS['latency']}, Sync={DEFAULT_WEIGHTS['sync']}, Stake={DEFAULT_WEIGHTS['stake']}")
    print("-" * 100)
    print(f"{'Provider':<25} | {'Latency Contrib':<15} | {'Sync Contrib':<15} | {'Avail Contrib':<15} | {'Stake Contrib':<15}")
    print("-" * 100)
    
    for provider, p_stats in sorted_providers:
        count = len(p_stats['history_score'])
        if count == 0: continue
        
        avg_lat = sum(p_stats['history_latency']) / count
        avg_sync = sum(p_stats['history_sync']) / count
        avg_avail = sum(p_stats['history_avail']) / count
        avg_stake = sum(p_stats['history_stake']) / count
        
        c_lat = avg_lat * DEFAULT_WEIGHTS['latency']
        c_sync = avg_sync * DEFAULT_WEIGHTS['sync']
        c_avail = avg_avail * DEFAULT_WEIGHTS['availability']
        c_stake = avg_stake * DEFAULT_WEIGHTS['stake']
        
        print(f"{provider:<25} | {c_lat:.4f} ({c_lat/avg_score*100:4.1f}%) | {c_sync:.4f} ({c_sync/avg_score*100:4.1f}%) | {c_avail:.4f} ({c_avail/avg_score*100:4.1f}%) | {c_stake:.4f} ({c_stake/avg_score*100:4.1f}%)")
    print("=" * 100)
    
    # --- Graph Output ---
    if HAS_MATPLOTLIB and output_graph:
        print(f"\nGenerating graphs...")
        print(f"  - Main analysis: {output_graph}")
        if skip_initial_events > 0:
            print(f"  - Skipping first {skip_initial_events} events in QoS charts to avoid warmup period")
        generate_graph(stats, providers_list, actual_pcts, exp_pcts, output_graph, skip_initial_events)
    elif output_graph:
        print("\nWarning: matplotlib not installed, cannot generate graph.")
    
    return stats

def print_per_request_details(stats, max_requests=10):
    """Print detailed information about individual requests"""
    if not stats['by_guid']:
        print("\nNo GUID data available for per-request analysis")
        return
    
    print(f"\n{'='*80}")
    print(f"PER-REQUEST DETAILS (showing up to {max_requests} requests)")
    print(f"{'='*80}")
    
    # Sort by number of selections (most attempts first) to show problematic requests
    sorted_guids = sorted(stats['by_guid'].items(), 
                          key=lambda x: len(x[1]), 
                          reverse=True)
    
    for idx, (guid, selections) in enumerate(sorted_guids[:max_requests]):
        print(f"\nRequest #{idx+1} - GUID: {guid}")
        print(f"  Total Attempts: {len(selections)}")
        
        for i, sel in enumerate(selections, 1):
            print(f"  Attempt {i}:")
            print(f"    Selected Provider: {sel['selected_provider']}")
            print(f"    Timestamp: {sel['timestamp']}")
            print(f"    Candidates: {sel['num_candidates']}")
            print(f"    Total Score: {sel['total_score']:.4f}")
            print(f"    Random Value: {sel['random_value']:.4f}")
        
        # If multiple attempts, show provider switching
        if len(selections) > 1:
            providers = [s['selected_provider'] for s in selections]
            unique_providers = len(set(providers))
            print(f"  Provider Switches: {len(selections)-1} attempts, {unique_providers} unique provider(s)")
            if unique_providers < len(selections):
                print(f"  Note: Same provider selected multiple times (possible retry)")

def generate_graph(stats, providers, actual_pcts, exp_pcts, filename, skip_initial_events=0):
    # Create main overview graph
    fig = plt.figure(figsize=(15, 15))
    gs = gridspec.GridSpec(3, 2, height_ratios=[1, 1, 1])

    # 1. Bar Chart: Actual vs Expected
    ax1 = plt.subplot(gs[0, :])
    x = range(len(providers))
    width = 0.35
    
    ax1.bar([i - width/2 for i in x], actual_pcts, width, label='Actual Selection %', color='skyblue')
    ax1.bar([i + width/2 for i in x], exp_pcts, width, label='Expected Probability %', color='orange', alpha=0.7)
    
    ax1.set_ylabel('Percentage')
    ax1.set_title('Provider Selection: Actual vs Expected')
    ax1.set_xticks(x)
    ax1.set_xticklabels(providers)
    ax1.legend()
    ax1.grid(axis='y', linestyle='--', alpha=0.3)

    # 2. Time Series: Scores
    ax2 = plt.subplot(gs[1, 0])
    for provider, p_stats in stats['providers'].items():
        if p_stats['history_score']:
            ax2.plot(p_stats['history_score'], label=provider, alpha=0.8)
    
    ax2.set_xlabel('Selection Event Index')
    ax2.set_ylabel('Score (Selection Weight)')
    ax2.set_title('Provider Scores Over Time')
    ax2.legend(fontsize='small')
    ax2.grid(True, linestyle='--', alpha=0.3)

    # 3. Box Plot: Score Distribution
    ax3 = plt.subplot(gs[1, 1])
    data_to_plot = []
    labels = []
    for provider, p_stats in sorted(stats['providers'].items()):
        if p_stats['history_score']:
            data_to_plot.append(p_stats['history_score'])
            labels.append(provider.replace('primary-', '').replace('-jsonrpc', ''))
    
    if data_to_plot:
        ax3.boxplot(data_to_plot, tick_labels=labels)
        ax3.set_ylabel('Score Distribution')
        ax3.set_title('Score Variance per Provider')
        ax3.grid(True, linestyle='--', alpha=0.3)
    else:
        ax3.text(0.5, 0.5, 'No score data available', ha='center', va='center', transform=ax3.transAxes)
        ax3.set_title('Score Variance per Provider')
        ax3.axis('off')

    # 4. Histogram: Normalized RNG Values
    ax4 = plt.subplot(gs[2, :])
    norm_rng_vals = stats['normalized_rng_values']
    if norm_rng_vals:
        ax4.hist(norm_rng_vals, bins=50, color='green', alpha=0.7, edgecolor='black')
        ax4.set_xlabel('Normalized RNG Value (Random / TotalScore)')
        ax4.set_ylabel('Frequency')
        ax4.set_title('Distribution of Normalized RNG (Should be Uniform [0, 1])')
        ax4.grid(True, linestyle='--', alpha=0.3)
        
        # Add mean line
        avg_norm = sum(norm_rng_vals) / len(norm_rng_vals)
        ax4.axvline(avg_norm, color='red', linestyle='dashed', linewidth=1, label=f'Mean: {avg_norm:.4f}')
        ax4.legend()

    plt.tight_layout()
    plt.savefig(filename)
    print(f"Graph saved to {filename}")
    
    # Create QoS breakdown comparison chart
    generate_qos_comparison_chart(stats, filename, skip_initial_events)
    
    # Create per-provider all-params chart
    generate_per_provider_charts(stats, filename, skip_initial_events)
    
    # Create separate QoS parameter charts
    generate_qos_charts(stats, filename, skip_initial_events)

def generate_per_provider_charts(stats, base_filename, skip_initial_events=0):
    """Generate charts showing all QoS parameters for each provider with P10, P90, and Average."""
    
    base_name = base_filename.rsplit('.', 1)[0]
    
    # QoS parameters to plot
    qos_params = [
        ('latency', 'Latency Score', 'history_latency', 'blue'),
        ('sync', 'Sync Score', 'history_sync', 'green'),
        ('availability', 'Availability Score', 'history_avail', 'orange'),
        ('stake', 'Stake Score', 'history_stake', 'purple')
    ]
    
    for provider, p_stats in sorted(stats['providers'].items()):
        # Create figure with 2x2 grid for 4 QoS parameters
        fig, axes = plt.subplots(2, 2, figsize=(16, 12))
        axes = axes.flatten()
        
        provider_short = provider.replace('primary-', '').replace('-jsonrpc', '')
        
        has_data = False
        
        for idx, (param_key, param_name, history_key, color) in enumerate(qos_params):
            ax = axes[idx]
            
            if p_stats[history_key]:
                data = p_stats[history_key][skip_initial_events:] if skip_initial_events > 0 else p_stats[history_key]
                
                if len(data) > 0:
                    has_data = True
                    
                    # Calculate statistics
                    sorted_data = sorted(data)
                    n = len(sorted_data)
                    
                    # P10, P90, Average
                    p10_idx = int(n * 0.10)
                    p90_idx = int(n * 0.90)
                    p10 = sorted_data[p10_idx] if p10_idx < n else sorted_data[0]
                    p90 = sorted_data[p90_idx] if p90_idx < n else sorted_data[-1]
                    avg = sum(data) / len(data)
                    
                    # Plot time series
                    ax.plot(data, color=color, alpha=0.6, linewidth=1.5, label='Score')
                    
                    # Plot P10, P90, Average as horizontal lines
                    ax.axhline(p10, color='red', linestyle='--', linewidth=1.5, 
                              label=f'P10: {p10:.3f}', alpha=0.8)
                    ax.axhline(avg, color='black', linestyle='-', linewidth=2, 
                              label=f'Avg: {avg:.3f}', alpha=0.8)
                    ax.axhline(p90, color='darkgreen', linestyle='--', linewidth=1.5, 
                              label=f'P90: {p90:.3f}', alpha=0.8)
                    
                    # Fill area between P10 and P90
                    ax.fill_between(range(len(data)), p10, p90, alpha=0.15, color=color)
                    
                    # Formatting
                    ax.set_xlabel('Selection Event Index', fontsize=10)
                    ax.set_ylabel('Normalized Score', fontsize=10)
                    ax.set_title(f'{param_name}', fontsize=12, fontweight='bold')
                    ax.legend(fontsize=9, loc='best')
                    ax.grid(True, linestyle='--', alpha=0.3)
                    
                    # Auto-scale y-axis if data range is small
                    data_min = min(data)
                    data_max = max(data)
                    data_range = data_max - data_min
                    
                    # If range is very small (< 0.15), zoom in to show detail
                    if data_range < 0.15:
                        # Add 10% padding above and below
                        padding = max(0.02, data_range * 0.1)
                        y_min = max(0, data_min - padding)
                        y_max = min(1.0, data_max + padding)
                        ax.set_ylim([y_min, y_max])
                    else:
                        # Use full 0-1 range for larger variations
                        ax.set_ylim([0, 1.05])
                    
                    # Add weight info
                    weight = DEFAULT_WEIGHTS[param_key]
                    contribution = avg * weight
                    ax.text(0.98, 0.02, f'Weight: {weight:.1f} | Contrib: {contribution:.3f}',
                           transform=ax.transAxes, fontsize=9, 
                           verticalalignment='bottom', horizontalalignment='right',
                           bbox=dict(boxstyle='round', facecolor='wheat', alpha=0.5))
        
        if has_data:
            fig.suptitle(f'{provider_short} - QoS Parameters Analysis (P10/Avg/P90)', 
                        fontsize=14, fontweight='bold', y=0.995)
            plt.tight_layout(rect=[0, 0, 1, 0.99])
            output_file = f"{base_name}_provider_{provider_short}.png"
            plt.savefig(output_file)
            print(f"  - {provider_short} comprehensive chart saved to {output_file}")
            plt.close()
        else:
            plt.close()

def generate_qos_comparison_chart(stats, base_filename, skip_initial_events=0):
    """Generate comparison chart showing normalized scores vs weighted contributions."""
    
    base_name = base_filename.rsplit('.', 1)[0]
    
    # QoS parameters with their weights
    qos_params = [
        ('latency', 'Latency', 'history_latency', DEFAULT_WEIGHTS['latency']),
        ('sync', 'Sync', 'history_sync', DEFAULT_WEIGHTS['sync']),
        ('availability', 'Availability', 'history_avail', DEFAULT_WEIGHTS['availability']),
        ('stake', 'Stake', 'history_stake', DEFAULT_WEIGHTS['stake'])
    ]
    
    sorted_providers = sorted(stats['providers'].items())
    
    # Create a figure with subplots: one row per provider
    num_providers = len(sorted_providers)
    fig, axes = plt.subplots(num_providers, 1, figsize=(16, 5 * num_providers))
    
    # Handle single provider case
    if num_providers == 1:
        axes = [axes]
    
    for idx, (provider, p_stats) in enumerate(sorted_providers):
        ax = axes[idx]
        
        # Calculate average normalized scores and contributions
        param_names = []
        normalized_scores = []
        weighted_contributions = []
        
        for param_key, param_name, history_key, weight in qos_params:
            if p_stats[history_key]:
                data = p_stats[history_key][skip_initial_events:] if skip_initial_events > 0 else p_stats[history_key]
                if len(data) > 0:
                    avg_normalized = sum(data) / len(data)
                    avg_contribution = avg_normalized * weight
                    
                    param_names.append(param_name)
                    normalized_scores.append(avg_normalized)
                    weighted_contributions.append(avg_contribution)
        
        # Create grouped bar chart
        x = range(len(param_names))
        width = 0.35
        
        bars1 = ax.bar([i - width/2 for i in x], normalized_scores, width, 
                       label='Normalized Score (0-1)', color='skyblue', alpha=0.8)
        bars2 = ax.bar([i + width/2 for i in x], weighted_contributions, width,
                       label='Weighted Contribution', color='coral', alpha=0.8)
        
        # Add value labels on bars
        for bar in bars1:
            height = bar.get_height()
            ax.text(bar.get_x() + bar.get_width()/2., height,
                   f'{height:.3f}',
                   ha='center', va='bottom', fontsize=9)
        
        for bar in bars2:
            height = bar.get_height()
            ax.text(bar.get_x() + bar.get_width()/2., height,
                   f'{height:.3f}',
                   ha='center', va='bottom', fontsize=9)
        
        ax.set_ylabel('Score', fontsize=11)
        ax.set_title(f'{provider.replace("primary-", "").replace("-jsonrpc", "")} - QoS Breakdown', 
                    fontsize=13, fontweight='bold')
        ax.set_xticks(x)
        ax.set_xticklabels(param_names)
        ax.legend(fontsize=10)
        ax.grid(True, linestyle='--', alpha=0.3, axis='y')
        ax.set_ylim([0, 1.1])
    
    plt.tight_layout()
    output_file = f"{base_name}_qos_breakdown.png"
    plt.savefig(output_file)
    print(f"  - QoS breakdown chart saved to {output_file}")
    plt.close()

def generate_qos_charts(stats, base_filename, skip_initial_events=0):
    """Generate separate charts for each QoS parameter."""
    
    # Get base filename without extension
    base_name = base_filename.rsplit('.', 1)[0]
    
    # Define QoS parameters to chart
    qos_params = [
        ('latency', 'Latency Score', 'history_latency', 'blue'),
        ('sync', 'Sync Score', 'history_sync', 'green'),
        ('availability', 'Availability Score', 'history_avail', 'orange'),
        ('stake', 'Stake Score', 'history_stake', 'purple')
    ]
    
    for param_key, param_title, history_key, color in qos_params:
        # Create a figure with 2 subplots: time series and box plot
        fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(15, 6))
        
        # Time Series Chart
        has_data = False
        all_data_combined = []
        for provider, p_stats in stats['providers'].items():
            if p_stats[history_key]:
                # Skip initial warmup events if specified
                data = p_stats[history_key][skip_initial_events:] if skip_initial_events > 0 else p_stats[history_key]
                if len(data) > 0:
                    ax1.plot(data, label=provider, alpha=0.8, linewidth=2)
                    all_data_combined.extend(data)
                    has_data = True
        
        if has_data:
            ax1.set_xlabel('Selection Event Index', fontsize=11)
            ax1.set_ylabel(f'{param_title} (Normalized)', fontsize=11)
            ax1.set_title(f'{param_title} Over Time', fontsize=13, fontweight='bold')
            ax1.legend(fontsize='small', loc='best')
            ax1.grid(True, linestyle='--', alpha=0.3)
            
            # Auto-scale y-axis if data range is small
            if all_data_combined:
                data_min = min(all_data_combined)
                data_max = max(all_data_combined)
                data_range = data_max - data_min
                
                # If range is very small (< 0.15), zoom in to show detail
                if data_range < 0.15:
                    padding = max(0.02, data_range * 0.1)
                    y_min = max(0, data_min - padding)
                    y_max = min(1.0, data_max + padding)
                    ax1.set_ylim([y_min, y_max])
                else:
                    ax1.set_ylim([0, 1.05])
            
            # Box Plot: Distribution
            data_to_plot = []
            labels = []
            for provider, p_stats in sorted(stats['providers'].items()):
                if p_stats[history_key]:
                    # Skip initial warmup events if specified
                    data = p_stats[history_key][skip_initial_events:] if skip_initial_events > 0 else p_stats[history_key]
                    if len(data) > 0:
                        data_to_plot.append(data)
                        labels.append(provider.replace('primary-', '').replace('-jsonrpc', ''))
            
            bp = ax2.boxplot(data_to_plot, tick_labels=labels, patch_artist=True)
            
            # Color the boxes
            for patch in bp['boxes']:
                patch.set_facecolor(color)
                patch.set_alpha(0.6)
            
            ax2.set_ylabel(f'{param_title} (Normalized)', fontsize=11)
            ax2.set_title(f'{param_title} Distribution', fontsize=13, fontweight='bold')
            ax2.grid(True, linestyle='--', alpha=0.3, axis='y')
            
            # Auto-scale y-axis if data range is small
            all_values = [v for data in data_to_plot for v in data]
            if all_values:
                data_min = min(all_values)
                data_max = max(all_values)
                data_range = data_max - data_min
                
                # If range is very small (< 0.15), zoom in to show detail
                if data_range < 0.15:
                    padding = max(0.02, data_range * 0.1)
                    y_min = max(0, data_min - padding)
                    y_max = min(1.0, data_max + padding)
                    ax2.set_ylim([y_min, y_max])
                else:
                    ax2.set_ylim([0, 1.05])
                
                # Add horizontal line at mean
                mean_val = sum(all_values) / len(all_values)
                ax2.axhline(mean_val, color='red', linestyle='--', linewidth=1, 
                           label=f'Mean: {mean_val:.4f}', alpha=0.7)
                ax2.legend(fontsize='small')
            
            plt.tight_layout()
            output_file = f"{base_name}_{param_key}.png"
            plt.savefig(output_file)
            print(f"  - {param_title} chart saved to {output_file}")
            plt.close()
        else:
            plt.close()
            print(f"  - No data for {param_title}, skipping chart")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Analyze WRS logs with GUID tracking")
    parser.add_argument("log_file", nargs="?", default="testutil/debugging/logs/CONSUMER.log", help="Path to log file")
    parser.add_argument("--graph", "-g", help="Output path for graph image (e.g. wrs_analysis.png)", default="wrs_analysis.png")
    parser.add_argument("--skip-initial", type=int, default=500, help="Skip first N events in QoS charts to avoid warmup period (default: 500)")
    parser.add_argument("--show-requests", "-r", type=int, nargs="?", const=10, default=0, 
                       help="Show detailed per-request analysis. Optionally specify max number of requests to show (default: 10)")
    parser.add_argument("--json", action="store_true", help="Parse log file as JSON format (each line is a JSON object)")
    
    args = parser.parse_args()
    
    stats = analyze_logs(args.log_file, args.graph, args.skip_initial, json_format=args.json)
    
    # Show per-request details if requested
    if args.show_requests > 0:
        print_per_request_details(stats, args.show_requests)
