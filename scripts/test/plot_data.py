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

def plot_graphs(data, provider, output_file):
    timestamps = [parser.parse(entry['timestamp']) for entry in data]
    sync_scores = [float(entry['sync_score']) for entry in data]
    availability_scores = [float(entry['availability_score']) for entry in data]
    latency_scores = [float(entry['latency_score']) for entry in data]
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
    plt.title(f'Provider: {provider}, Avg Node Error Rate: {avg_node_error_rate:.2f}, Stake: {provider_stake}')
    plt.legend()
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(output_file)
    print(f"Plot saved as {output_file}")

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print("Usage: python plot_data.py <current_mechanism_csv> <refactored_mechanism_csv>")
        sys.exit(1)

    current_csv_file = sys.argv[1]
    refactored_csv_file = sys.argv[2]

    current_data = read_csv(current_csv_file)
    refactored_data = read_csv(refactored_csv_file)

    current_grouped_data = group_by_provider(current_data)
    refactored_grouped_data = group_by_provider(refactored_data)

    # Randomly select one provider from the current mechanism data
    selected_provider = random.choice(list(current_grouped_data.keys()))

    # Ensure the selected provider exists in both datasets
    if selected_provider not in refactored_grouped_data:
        print(f"Provider {selected_provider} not found in refactored data.")
        sys.exit(1)

    current_selected_data = current_grouped_data[selected_provider]
    refactored_selected_data = refactored_grouped_data[selected_provider]

    # Extract the last 4 characters of the provider address
    provider_suffix = selected_provider.split('@')[-1][-4:]

    # Plot graphs for both datasets
    plot_graphs(current_selected_data, selected_provider, f"provider_{provider_suffix}_current.png")
    plot_graphs(refactored_selected_data, selected_provider, f"provider_{provider_suffix}_refactor.png") 