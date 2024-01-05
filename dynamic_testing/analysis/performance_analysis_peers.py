import json
import numpy as np
import matplotlib.pyplot as plt
from datetime import datetime
from dateutil.parser import parse as parse_dateutil

# Function to parse ISO 8601 timestamps
def parse_timestamp(timestamp):
    return parse_dateutil(timestamp)

# Function to calculate mean and standard deviation, handling None values
def calculate_mean_std(values):
    filtered_values = [v for v in values if v is not None]
    if filtered_values:
        mean_val = np.mean(filtered_values)
        std_val = np.std(filtered_values)
    else:
        mean_val, std_val = None, None
    return mean_val, std_val

# Function to process single repair data
def process_single_repair_data(data):
    #only filter the depth 7 data
    processed_data = {}
    for failed_peers, experiments in data['depth_7'].items():
        if int(failed_peers) <= 7:  # Consider only failed peers from 1 to 7
            successful_attempts = [exp for exp in experiments if exp['status'] == 0]
            times = [(parse_timestamp(exp['endTime']) - parse_timestamp(exp['startTime'])).total_seconds() for exp in successful_attempts]
            blocks_downloaded = [exp['dataBlocksFetched'] + exp['parityBlocksFetched'] for exp in successful_attempts]
            processed_data[failed_peers] = {
                'average_time': calculate_mean_std(times)[0],
                'average_blocks_downloaded': calculate_mean_std(blocks_downloaded)[0],
                'std_time': calculate_mean_std(times)[1],
                'std_blocks_downloaded': calculate_mean_std(blocks_downloaded)[1]
            }
    return processed_data

# Function to process collaborative repair data
def process_collab_repair_data(data):
    repair_peers_variations = [3,5,7,9]
    processed_data = {}
    for repair_peers in repair_peers_variations:
        processed_data[repair_peers] = {}
        for failed_peers, experiments in data[f"repair_peers_{repair_peers}"].items():
            if int(failed_peers) <= 7:  # Consider only failed peers from 1 to 7
                filtered_experiments = []
                for exp in experiments:
                    status = exp['metrics']['status']
                    sum_blocks = sum([1 for _, peerData in  exp['metrics']['peers'].items() for _, downloaded in peerData['blocks'].items() if downloaded])
                    if status == 1 or (status == 2 and sum_blocks >= 90):
                        filtered_experiments.append(exp)
                total_times = [(parse_timestamp(exp['metrics']['endTime']) - parse_timestamp(exp['metrics']['startTime'])).total_seconds() for exp in filtered_experiments]
                # peer_times = [np.mean([(parse_timestamp(peer['endTime']) - parse_timestamp(peer['startTime'])).total_seconds() for peer in exp['metrics']['peers'].values()]) for exp in filtered_experiments]
                peer_times = [(parse_timestamp(peer['endTime']) - parse_timestamp(peer['startTime'])).total_seconds() for exp in filtered_experiments for peer in exp['metrics']['peers'].values() ]
                blocks_downloaded = [sum(peer['dataBlocksFetched'] + peer['parityBlocksFetched'] for peer in exp['metrics']['peers'].values()) for exp in filtered_experiments]
                processed_data[repair_peers][failed_peers] = {
                    'average_total_time': calculate_mean_std(total_times)[0],
                    'average_peer_time': calculate_mean_std(peer_times)[0],
                    'average_total_blocks_fetched': calculate_mean_std(blocks_downloaded)[0],
                    'average_blocks_per_peer': calculate_mean_std([peer['dataBlocksFetched'] + peer['parityBlocksFetched'] for exp in filtered_experiments for peer in exp['metrics']['peers'].values()])[0],
                    'std_total_time': calculate_mean_std(total_times)[1],
                    'std_peer_time': calculate_mean_std(peer_times)[1],
                    'std_total_blocks_fetched': calculate_mean_std(blocks_downloaded)[1],
                    'std_blocks_per_peer': calculate_mean_std([peer['dataBlocksFetched'] + peer['parityBlocksFetched'] for exp in filtered_experiments for peer in exp['metrics']['peers'].values()])[1]
                }
    return processed_data

def generate_and_save_plots(repair_peers, single_repair_data, collab_repair_data):
    failed_peers = list(range(1, 8, 2))  # Considering failed peers from 1 to 7
    plot_colors = {'single_repair': 'red', 'collab_repair': 'blue'}

    # Defining the metrics for comparison
    plot_metrics = [
        ('average_time', 'average_total_time', 'Average Repair Time Comparison', 'Average Time(s)'),
        ('average_time', 'average_peer_time', 'Average Time Per Peer Comparison', 'Average Time(s)'),
        ('average_blocks_downloaded', 'average_total_blocks_fetched', 'Total Blocks Downloaded Comparison' , 'No. Blocks Downloaded'),
        ('average_blocks_downloaded', 'average_blocks_per_peer', 'Blocks Downloaded Per Peer Comparison', 'No. Blocks Downloaded')
    ]

    for plot_index, (single_metric, collab_metric, title, y_label) in enumerate(plot_metrics):
        plt.figure()
        # Extracting data for plotting
        single_means = [single_repair_data.get(str(peer), {}).get(single_metric, np.nan) for peer in failed_peers]
        single_stds = [single_repair_data.get(str(peer), {}).get(f'std_{single_metric.split("_")[1]}', 0) for peer in failed_peers]
        collab_means = [collab_repair_data.get(repair_peers, {}).get(str(peer), {}).get(collab_metric, np.nan) for peer in failed_peers]
        collab_stds = [collab_repair_data.get(repair_peers, {}).get(str(peer), {}).get(f'std_{collab_metric.split("_")[1]}', 0) for peer in failed_peers]

        # Plotting lines for mean values
        plt.plot(failed_peers, single_means, '-', color=plot_colors['single_repair'], label='Single Repair')
        plt.plot(failed_peers, collab_means, '-', color=plot_colors['collab_repair'], label='Collaborative Repair')

        # Adding shaded regions for standard deviation
        plt.fill_between(failed_peers, np.array(single_means) - np.array(single_stds), np.array(single_means) + np.array(single_stds), color=plot_colors['single_repair'], alpha=0.1)
        plt.fill_between(failed_peers, np.array(collab_means) - np.array(collab_stds), np.array(collab_means) + np.array(collab_stds), color=plot_colors['collab_repair'], alpha=0.1)

        plt.xlabel('Number of Failed Peers')
        plt.ylabel(y_label)
        plt.title(f'Repair Peers: {repair_peers} - {title}')
        plt.legend()
        plt.savefig(f'./plots/performance/plot_repair_peers_{repair_peers}_{plot_index + 1}.png')



def generate_collab_comparison_plots(collab_repair_data):
    failed_peers = list(range(1, 8, 2))  # Considering failed peers from 1 to 7
    plot_colors = ['blue', 'green', 'orange', 'purple', 'brown', 'pink', 'grey']

    # Defining the metrics for comparison
    plot_metrics = [
        ('average_total_time', 'Average Total Repair Time Comparison', 'Average Time(s)'),
        ('average_peer_time', 'Average Time Per Peer Comparison', 'Average Time(s)'),
        ('average_total_blocks_fetched', 'Total Blocks Downloaded Comparison', 'No. Blocks Downloaded'),
        ('average_blocks_per_peer', 'Blocks Downloaded Per Peer Comparison', 'No. Blocks Downloaded')
    ]

    # Defining different repair peer configurations
    repair_peers_configs = [3,5,7,9]

    for plot_index, (metric, title, y_label) in enumerate(plot_metrics):
        plt.figure()
        for i, repair_peers in enumerate(repair_peers_configs):
            # Extracting data for each repair peers configuration
            means = [collab_repair_data.get(repair_peers, {}).get(str(peer), {}).get(metric, np.nan) for peer in failed_peers]
            stds = [collab_repair_data.get(repair_peers, {}).get(str(peer), {}).get(f'std_{metric.split("_")[1]}', 0) for peer in failed_peers]

            # Plotting lines for mean values
            plt.plot(failed_peers, means, '-', color=plot_colors[i % len(plot_colors)], label=f'Collaborative Repair - {repair_peers} peers')
            # Adding shaded regions for standard deviation
            plt.fill_between(failed_peers, np.array(means) - np.array(stds), np.array(means) + np.array(stds), color=plot_colors[i % len(plot_colors)], alpha=0.1)

        plt.xlabel('Number of Failed Peers')
        plt.ylabel(y_label)
        plt.title(title)
        plt.legend()
        plt.savefig(f'./plots/performance/collab_comparison_{metric}.png')




# Load and process data from JSON files
file_paths = {
    'single_repair': '../results/results_performance_single_repair.json',
    'collab_repair': '../results/results_performance_collab_repair_peers.json'
}

data = {}
for key, file_path in file_paths.items():
    with open(file_path, 'r') as file:
        data[key] = json.load(file)

single_repair_processed = process_single_repair_data(data['single_repair'])
collab_repair_processed = process_collab_repair_data(data['collab_repair'])

# Generating and saving plots for each depth
repair_peers_variations = [5,7,9]
for repair_peers in repair_peers_variations:
    generate_and_save_plots(repair_peers, single_repair_processed, collab_repair_processed)

generate_collab_comparison_plots(collab_repair_processed)

# Confirmation message
print("Plots generated and saved for all depths.")
