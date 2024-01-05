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
    processed_data = {}
    for depth, depth_data in data.items():
        processed_data[depth] = {}
        for failed_peers, experiments in depth_data.items():
            if int(failed_peers) <= 7:  # Consider only failed peers from 1 to 7
                successful_attempts = [exp for exp in experiments if exp['status'] == 0]
                times = [(parse_timestamp(exp['endTime']) - parse_timestamp(exp['startTime'])).total_seconds() for exp in successful_attempts]
                blocks_downloaded = [exp['dataBlocksFetched'] + exp['parityBlocksFetched'] for exp in successful_attempts]
                processed_data[depth][failed_peers] = {
                    'average_time': calculate_mean_std(times)[0],
                    'average_blocks_downloaded': calculate_mean_std(blocks_downloaded)[0],
                    'std_time': calculate_mean_std(times)[1],
                    'std_blocks_downloaded': calculate_mean_std(blocks_downloaded)[1]
                }
    return processed_data

# Function to process collaborative repair data
def process_collab_repair_data(data):
    processed_data = {}
    for depth, depth_data in data.items():
        processed_data[depth] = {}
        for failed_peers, experiments in depth_data.items():
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
                processed_data[depth][failed_peers] = {
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

def generate_and_save_plots(depth, single_repair_data, collab_repair_data):
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
        single_means = [single_repair_data.get(depth, {}).get(str(peer), {}).get(single_metric, np.nan) for peer in failed_peers]
        single_stds = [single_repair_data.get(depth, {}).get(str(peer), {}).get(f'std_{single_metric.split("_")[1]}', 0) for peer in failed_peers]
        collab_means = [collab_repair_data.get(depth, {}).get(str(peer), {}).get(collab_metric, np.nan) for peer in failed_peers]
        collab_stds = [collab_repair_data.get(depth, {}).get(str(peer), {}).get(f'std_{collab_metric.split("_")[1]}', 0) for peer in failed_peers]

        # Plotting lines for mean values
        plt.plot(failed_peers, single_means, '-', color=plot_colors['single_repair'], label='Single Repair')
        plt.plot(failed_peers, collab_means, '-', color=plot_colors['collab_repair'], label='Collaborative Repair')

        # Adding shaded regions for standard deviation
        plt.fill_between(failed_peers, np.array(single_means) - np.array(single_stds), np.array(single_means) + np.array(single_stds), color=plot_colors['single_repair'], alpha=0.1)
        plt.fill_between(failed_peers, np.array(collab_means) - np.array(collab_stds), np.array(collab_means) + np.array(collab_stds), color=plot_colors['collab_repair'], alpha=0.1)

        plt.xlabel('Number of Failed Peers')
        plt.ylabel(y_label)
        plt.title(f'Depth {depth[6:]} - {title}')
        plt.legend()
        plt.savefig(f'./plots/performance/plot_depth_{depth[6:]}_{plot_index + 1}.png')


# def generate_and_save_plots(depth, single_repair_data, collab_repair_data):
#     failed_peers = list(range(1, 8))  # Considering failed peers from 1 to 7
#     plot_colors = {'single_repair': 'red', 'collab_repair': 'blue'}

#     # Defining the metrics for comparison
#     plot_metrics = [
#         ('average_time', 'average_total_time', 'Average Repair Time Comparison'),
#         ('average_time', 'average_peer_time', 'Average Time Per Peer Comparison'),
#         ('average_blocks_downloaded', 'average_total_blocks_fetched', 'Total Blocks Downloaded Comparison'),
#         ('average_blocks_downloaded', 'average_blocks_per_peer', 'Blocks Downloaded Per Peer Comparison')
#     ]

#     for plot_index, (single_metric, collab_metric, title) in enumerate(plot_metrics):
#         plt.figure()
#         # Extracting and plotting data for each failed peer
#         for peer in failed_peers:
#             single_data = single_repair_data.get(depth, {}).get(str(peer), {})
#             collab_data = collab_repair_data.get(depth, {}).get(str(peer), {})
#             single_mean = single_data.get(single_metric)
#             single_std = single_data.get(f'std_{single_metric.split("_")[1]}', 0)  # Default to 0 if None
#             collab_mean = collab_data.get(collab_metric)
#             collab_std = collab_data.get(f'std_{collab_metric.split("_")[1]}', 0)  # Default to 0 if None
            
#             if single_mean is not None and collab_mean is not None:
#                 plt.plot(peer, single_mean, 'o', color=plot_colors['single_repair'])
#                 plt.plot(peer, collab_mean, 'o', color=plot_colors['collab_repair'])
#                 plt.fill_between([peer], single_mean - single_std, single_mean + single_std, color=plot_colors['single_repair'], alpha=0.1)
#                 plt.fill_between([peer], collab_mean - collab_std, collab_mean + collab_std, color=plot_colors['collab_repair'], alpha=0.1)

#         # Interpolating missing data points
#         plt.plot(failed_peers, [single_repair_data.get(depth, {}).get(str(peer), {}).get(single_metric, np.nan) for peer in failed_peers], '-', color=plot_colors['single_repair'], label='Single Repair')
#         plt.plot(failed_peers, [collab_repair_data.get(depth, {}).get(str(peer), {}).get(collab_metric, np.nan) for peer in failed_peers], '-', color=plot_colors['collab_repair'], label='Collaborative Repair')

#         plt.xlabel('Number of Failed Peers')
#         plt.ylabel('Average ' + single_metric.replace('_', ' ').capitalize())
#         plt.title(f'Depth {depth[6:]} - {title}')
#         plt.legend()
#         plt.savefig(f'./plots/performance/plot_depth_{depth[6:]}_{plot_index + 1}.png')





# Load and process data from JSON files
file_paths = {
    'single_repair': '../results/results_performance_single_repair.json',
    'collab_repair': '../results/results_performance_collab_repair.json'
}

data = {}
for key, file_path in file_paths.items():
    with open(file_path, 'r') as file:
        data[key] = json.load(file)

single_repair_processed = process_single_repair_data(data['single_repair'])
collab_repair_processed = process_collab_repair_data(data['collab_repair'])

# Generating and saving plots for each depth
depths = ['depth_5', 'depth_7', 'depth_10']
for depth in depths:
    generate_and_save_plots(depth, single_repair_processed, collab_repair_processed)

# Confirmation message
print("Plots generated and saved for all depths.")
