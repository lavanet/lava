import csv
import matplotlib.pyplot as plt
from dateutil import parser
import sys
import random
from collections import defaultdict


def read_csv(file_path):
    data = []
    with open(file_path, 'r') as csvfile:
        reader = csv.DictReader(csvfile)
        for row in reader:
            data.append(row)
    return data


def group_by_provider(data):
    grouped_data = defaultdict(list)
    for entry in data:
        grouped_data[entry['provider']].append(entry)
    return grouped_data


def plot_graphs(data, provider, output_file, is_refactored=False):
    timestamps = [parser.parse(entry['timestamp']) for entry in data]
    sync_scores = [float(entry['sync_score']) for entry in data]
    availability_scores = [
        float(entry['availability_score'])
        for entry in data
        if entry['availability_score'] != '0'
    ]
    latency_scores = [
        float(entry['latency_score'])
        for entry in data
        if entry['latency_score'] != '0'
    ]
    generic_scores = [float(entry['generic_score']) for entry in data]
    node_error_rates = [float(entry['node_error_rate']) for entry in data]
    provider_stake = data[0]['provider_stake']  # Assuming stake is constant

    avg_node_error_rate = sum(node_error_rates) / len(node_error_rates)

    plt.figure(figsize=(10, 5))
    plt.plot(timestamps, sync_scores, label='Sync Score')
    plt.plot(timestamps, availability_scores, label='Availability Score')
    plt.plot(timestamps, latency_scores, label='Latency Score')
    plt.plot(timestamps, generic_scores, label='Generic Score')
    plt.xlabel('Timestamp')
    plt.ylabel('Scores')
    title_suffix = " (Refactor)" if is_refactored else ""
    plt.title(
        f'Provider: {provider}, Avg Node Err Rate: {avg_node_error_rate:.2f}, '
        f'Stake: {provider_stake}{title_suffix}'
    )
    plt.legend()
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(output_file)
    print(f"Plot saved as {output_file}")


def plot_tier_chances_over_time(data, provider, output_file):
    timestamps = [parser.parse(entry['timestamp']) for entry in data]
    tier_chances_dicts = [
        {int(k): float(v) for k, v in (
            item.split(': ') for item in
            entry['tier_chances'].split(', ') if item
        )}
        for entry in data
    ]

    # Collect all unique tiers
    all_tiers = set(tier for tier_chances in tier_chances_dicts
                    for tier in tier_chances.keys())

    plt.figure(figsize=(10, 5))
    for tier in sorted(all_tiers):
        tier_chances_over_time = [
            tier_chances.get(tier, 0)
            for tier_chances
            in tier_chances_dicts
        ]
        plt.plot(timestamps, tier_chances_over_time, label=f'Tier {tier}')

    plt.xlabel('Timestamp')
    plt.ylabel('Chance')
    plt.title(f'Provider: {provider} Tier Chances Over Time')
    plt.legend()
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(output_file)
    print(f"Tier chances over time plot saved as {output_file}")


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: python plot_data.py <csv_file_name>")
        sys.exit(1)

    csv_file_name = sys.argv[1]

    data = read_csv(csv_file_name)

    grouped_data = group_by_provider(data)

    # Randomly select one provider from the current mechanism data
    selected_provider = random.choice(list(grouped_data.keys()))

    selected_data = grouped_data[selected_provider]

    # Extract the last 4 characters of the provider address
    provider_suffix = selected_provider.split('@')[-1][-4:]

    # Plot graphs for both datasets
    plot_graphs(
        selected_data,
        selected_provider,
        f"provider_{provider_suffix}.png"
    )
    plot_tier_chances_over_time(
        selected_data,
        selected_provider,
        f"provider_{provider_suffix}_tier_chances.png"
    )
