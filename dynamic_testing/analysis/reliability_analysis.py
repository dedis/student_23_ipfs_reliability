
import json
import numpy as np
import matplotlib.pyplot as plt
import os 

def calculate_average_and_error(data):
    data_array = np.array(data)
    average = np.mean(data_array)
    error = np.std(data_array) / np.sqrt(len(data_array))
    return average, error

def prep_data(replication_data, reliability_data):
    graph_data = {}

    for num_peers in replication_data.keys():
        graph_data[num_peers] = {
            'replication': {},
            'erasure': {}
        }

        # For replication methods
        for replication_method in replication_data[num_peers].keys():
            failed_peers_list = list(map(int, replication_data[num_peers][replication_method].keys()))
            percentages = []
            full_block_percentages = []

            for failed_peers in failed_peers_list:
                blocks_downloaded = replication_data[num_peers][replication_method][str(failed_peers)]
                avg_blocks_downloaded, error = calculate_average_and_error(blocks_downloaded)
                percentages.append((avg_blocks_downloaded / 101, error))
                full_block_percentages.append((blocks_downloaded.count(101) / len(blocks_downloaded)))

            graph_data[num_peers]['replication'][replication_method] = {
                'failed_peers': failed_peers_list,
                'percentages': percentages,
                'full_block_percentages': full_block_percentages
            }

        # For AE Codes
        failed_peers_list = list(map(int, reliability_data[num_peers].keys()))
        percentages = []
        full_block_percentages = []

        for failed_peers in failed_peers_list:
            blocks_downloaded = reliability_data[num_peers][str(failed_peers)]
            avg_blocks_downloaded, error = calculate_average_and_error(blocks_downloaded)
            percentages.append((avg_blocks_downloaded / 101, error))
            full_block_percentages.append((blocks_downloaded.count(101) / len(blocks_downloaded)))

        graph_data[num_peers]['erasure'] = {
            'failed_peers': failed_peers_list,
            'percentages': percentages,
            'full_block_percentages': full_block_percentages
        }

    return graph_data

def plot_graph(prepared_data):
    colors = ['blue', 'red', 'green', 'black']
    for num_peers, data in prepared_data.items():
        replication_methods = list(data['replication'].keys())
        erasure_failed_peers = data['erasure']['failed_peers']

        # Graph 1: Percentage of blocks downloaded
        plt.figure(figsize=(15, 6))
        plt.title(f"Percentage of Blocks Downloaded vs Failed Peers (Peers: {num_peers})")
        plt.xlabel("Percentage of Failed Peers")
        plt.ylabel("Percentage of Blocks Downloaded")
        plt.ylim(0, 100)
        plt.grid(True)

        # Plot for each replication method
        for i, method in enumerate(replication_methods):
            failed_peers = data['replication'][method]['failed_peers']
            percentages = [p[0] for p in data['replication'][method]['percentages']]
            plt.plot(np.array(failed_peers) / int(num_peers) * 100, np.array(percentages) * 100, label=method, color=colors[i])
            

        # Plot for AE Codes
        erasure_percentages = [p[0] for p in data['erasure']['percentages']]
        plt.plot(np.array(erasure_failed_peers) / int(num_peers) * 100, np.array(erasure_percentages) * 100, label="AE Codes", color=colors[-1])
        plt.legend()
        # plt.show()
        plt.savefig(f'./plots/reliability/{num_peers}_peers_percentage_blocks_downloaded.png')

        # Graph 2: Probability of fully getting all blocks
        plt.figure(figsize=(15, 6))
        plt.title(f"Probability of Fully Getting All Blocks vs Failed Peers (Peers: {num_peers})")
        plt.xlabel("Percentage of Failed Peers")
        plt.ylabel("Probability of Fully Getting All Blocks")
        plt.ylim(0, 100)
        plt.grid(True)

        # Plot for each replication method
        for i, method in enumerate(replication_methods):
            failed_peers = data['replication'][method]['failed_peers']
            full_block_percentages = data['replication'][method]['full_block_percentages']
            plt.plot(np.array(failed_peers) / int(num_peers) * 100, np.array(full_block_percentages) * 100, label=method, color=colors[i])

        # Plot for AE Codes
        erasure_full_block_percentages = data['erasure']['full_block_percentages']
        plt.plot(np.array(erasure_failed_peers) / int(num_peers) * 100, np.array(erasure_full_block_percentages) * 100, label="AE Codes", color=colors[-1])
        plt.legend()
        # plt.show()
        plt.savefig(f'./plots/reliability/{num_peers}_peers_probability_fully_getting_all_blocks.png')

# Load the data
replication_path = os.path.abspath('../results/replication_results.json')
with open(replication_path, 'r') as file:
    replication_data = json.load(file)

reliability_path = os.path.abspath('../results/results_reliability.json')
with open(reliability_path, 'r') as file:
    reliability_data = json.load(file)

# Prepare data
prepared_data = prep_data(replication_data, reliability_data)

# Plot graphs without error bars
plot_graph(prepared_data)