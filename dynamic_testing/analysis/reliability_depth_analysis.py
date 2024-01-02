
import json
import numpy as np
import matplotlib.pyplot as plt
import os

# Function to calculate average and error
def calculate_average_and_error(data):
    data_array = np.array(data)
    average = np.mean(data_array)
    error = np.std(data_array) / np.sqrt(len(data_array))
    return average, error

# Preparing data for the new plots comparing replication (peers=20) to each repair depth in erasure codes

def prepare_data_for_repair_depth_comparison(replication_data, reliability_repair_data, num_peers=20):
    """
    Prepare data for plotting graphs that compare replication methods to different repair depths in erasure codes.
    """
    graph_data = {
        'replication': {},
        'erasure': {}
    }

    # Prepare replication data (only for peers=20)
    for replication_method in replication_data[str(num_peers)].keys():
        failed_peers_list = list(map(int, replication_data[str(num_peers)][replication_method].keys()))
        percentages = []
        full_block_percentages = []

        for failed_peers in failed_peers_list:
            blocks_downloaded = replication_data[str(num_peers)][replication_method][str(failed_peers)]
            avg_blocks_downloaded, error = calculate_average_and_error(blocks_downloaded)
            percentages.append((avg_blocks_downloaded / 101, error))
            full_block_percentages.append((blocks_downloaded.count(101) / len(blocks_downloaded)))

        graph_data['replication'][replication_method] = {
            'failed_peers': failed_peers_list,
            'percentages': percentages,
            'full_block_percentages': full_block_percentages
        }

    # Prepare erasure codes data for different repair depths
    for depth_key in reliability_repair_data.keys():
        failed_peers_list = list(map(int, reliability_repair_data[depth_key].keys()))
        percentages = []
        full_block_percentages = []

        for failed_peers in failed_peers_list:
            blocks_downloaded = reliability_repair_data[depth_key][str(failed_peers)]
            avg_blocks_downloaded, error = calculate_average_and_error(blocks_downloaded)
            percentages.append((avg_blocks_downloaded / 101, error))
            full_block_percentages.append((blocks_downloaded.count(101) / len(blocks_downloaded)))

        graph_data['erasure'][depth_key] = {
            'failed_peers': failed_peers_list,
            'percentages': percentages,
            'full_block_percentages': full_block_percentages
        }

    return graph_data

# Updating the plotting function to use the same colors as before and adding plots that compare different depths together

def plot_graphs_for_repair_depth_comparison_updated(prepared_data_for_repair_depth):
    """
    Plot graphs comparing replication (peers=20) replication 3,5,7 to each depth with updated colors.
    Also, plot two additional graphs comparing different depths together.
    """
    colors = ['blue', 'red', 'green', 'black']
    replication_methods = list(prepared_data_for_repair_depth['replication'].keys())
    erasure_depths = list(prepared_data_for_repair_depth['erasure'].keys())

    # Plot individual graphs for each depth
    for depth in erasure_depths:
        # Graph 1: Percentage of blocks downloaded
        plt.figure(figsize=(15, 6))
        plt.title(f"Percentage of Blocks Downloaded vs Failed Peers (AE-Codes Depth: {depth})")
        plt.xlabel("Percentage of Failed Peers")
        plt.ylabel("Percentage of Blocks Downloaded")
        plt.ylim(0, 100)
        plt.grid(True)

        # Plot for each replication method
        for i, method in enumerate(replication_methods):
            failed_peers = prepared_data_for_repair_depth['replication'][method]['failed_peers']
            percentages = [p[0] for p in prepared_data_for_repair_depth['replication'][method]['percentages']]
            plt.plot(np.array(failed_peers) / 20 * 100, np.array(percentages) * 100, label=method, color=colors[i])

        # Plot for erasure codes
        erasure_failed_peers = prepared_data_for_repair_depth['erasure'][depth]['failed_peers']
        erasure_percentages = [p[0] for p in prepared_data_for_repair_depth['erasure'][depth]['percentages']]
        plt.plot(np.array(erasure_failed_peers) / 20 * 100, np.array(erasure_percentages) * 100, label=f"AE-{depth}", color=colors[len(replication_methods)])
        plt.legend()
        plt.savefig(f'./plots/reliability_depth/ae_{depth}_peers_percentage_blocks_downloaded.png')

        # Graph 2: Probability of fully getting all blocks
        plt.figure(figsize=(15, 6))
        plt.title(f"Probability of Fully Getting All Blocks vs Failed Peers (AE Codes Depth: {depth})")
        plt.xlabel("Percentage of Failed Peers")
        plt.ylabel("Probability of Fully Getting All Blocks")
        plt.ylim(0, 100)
        plt.grid(True)

        # Plot for each replication method
        for i, method in enumerate(replication_methods):
            failed_peers = prepared_data_for_repair_depth['replication'][method]['failed_peers']
            full_block_percentages = prepared_data_for_repair_depth['replication'][method]['full_block_percentages']
            plt.plot(np.array(failed_peers) / 20 * 100, np.array(full_block_percentages) * 100, label=method, color=colors[i])

        # Plot for erasure codes
        erasure_full_block_percentages = prepared_data_for_repair_depth['erasure'][depth]['full_block_percentages']
        plt.plot(np.array(erasure_failed_peers) / 20 * 100, np.array(erasure_full_block_percentages) * 100, label=f"Erasure {depth}", color=colors[len(replication_methods)])
        plt.legend()
        plt.savefig(f'./plots/reliability_depth/ae_{depth}_peers_probability_fully_getting_all_blocks.png')

    # Additional Graphs comparing different depths together
    # Graph 1: Percentage of blocks downloaded for all depths
    plt.figure(figsize=(15, 6))
    plt.title("Percentage of Blocks Downloaded vs Failed Peers (Comparing Repair Depths)")
    plt.xlabel("Percentage of Failed Peers")
    plt.ylabel("Percentage of Blocks Downloaded")
    plt.ylim(0, 100)
    plt.grid(True)

    # Plot for each depth
    for i, depth in enumerate(erasure_depths):
        erasure_failed_peers = prepared_data_for_repair_depth['erasure'][depth]['failed_peers']
        erasure_percentages = [p[0] for p in prepared_data_for_repair_depth['erasure'][depth]['percentages']]
        plt.plot(np.array(erasure_failed_peers) / 20 * 100, np.array(erasure_percentages) * 100, label=f"AE-{depth}", color=colors[i])
    plt.legend()
    plt.savefig(f'./plots/reliability_depth/ae_depths_percentage_blocks_downloaded.png')

    # Graph 2: Probability of fully getting all blocks for all depths
    plt.figure(figsize=(15, 6))
    plt.title("Probability of Fully Getting All Blocks vs Failed Peers (Comparing Repair Depths)")
    plt.xlabel("Percentage of Failed Peers")
    plt.ylabel("Probability of Fully Getting All Blocks")
    plt.ylim(0, 100)
    plt.grid(True)

    # Plot for each depth
    for i, depth in enumerate(erasure_depths):
        erasure_failed_peers = prepared_data_for_repair_depth['erasure'][depth]['failed_peers']
        erasure_full_block_percentages = prepared_data_for_repair_depth['erasure'][depth]['full_block_percentages']
        plt.plot(np.array(erasure_failed_peers) / 20 * 100, np.array(erasure_full_block_percentages) * 100, label=f"AE-{depth}", color=colors[i])
    plt.legend()
    plt.savefig(f'./plots/reliability_depth/ae_depths_probability_fully_getting_all_blocks.png')



# Load the data
replication_path = os.path.abspath('../results/replication_results.json')
with open(replication_path, 'r') as file:
    replication_data = json.load(file)

reliability_path = os.path.abspath('../results/results_reliability_repair.json')
with open(reliability_path, 'r') as file:
    reliability_repair_data = json.load(file)
    
# Prepare data
prepared_data_for_repair_depth = prepare_data_for_repair_depth_comparison(replication_data, reliability_repair_data)

# Plot updated graphs
plot_graphs_for_repair_depth_comparison_updated(prepared_data_for_repair_depth)
