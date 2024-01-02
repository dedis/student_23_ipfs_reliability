from helpers import *
from test_data import test_data
import time 
import json

'''

we are testing the total number of blocks download/repaired with varying number of failed peers
defaults:
    - 10 peers
    - 4 replication factor
    - 7 repair depth
    - 25MB file
-> first test: Number of peers: 10, failed peers: 1 - 8
-> second test: Number of peers: 5, 10, 20, 40 -- failed peers: min 1, max N - 2
-> third test: Number of peers: 10, failed peers: 1 - 8, recovery depth: (3, 5, 7, 10)

as opposed to replication 
    replication factors: 3, 5, 7
-> first test: number of peers 10, failed peers 1 - 8
-> second test: number of peers 5, 10, 20, 40 -- failed peers: min 1, max N - 2
-> third test: number of peers 10, failed peers 1 - 8

comparison would be as follows:
-> first test
    - for each number of failed peers, 
        compare the number of blocks downloaded/repaired (repair, replication3, replication5, replication7)

-> second test (Actually includes the first test)
    - for each number of peers, A NEW GRAPH
        for each number of failed peers,
            compare the number of blocks downloaded/repaired (repair, replication3, replication5, replication7)

-> third test
    - for each number of recovery depth (3, 5, 7, 10)
        for each number of failed peers, 
            compare the number of blocks downloaded/repaired (repair, replication3, replication5, replication7

            
Technically for replication you just need the following:
    for numPeers in [10, 15, 20, 25, 30]:
        for replicationFactor in [3, 5, 7]:
            for failedPeers in range(1, numPeers - 2):
                runTest(numPeers, failedPeers, replicationFactor)

For repair we need to sets of test 
    for numPeers in [10, 15, 20, 25, 30]:
        for failedPeers in range(1, numPeers - 2):
                runTest(numPeers, failedPeers)
    
    for repairDepth in [3, 5, 7, 10]:
        for failedPeers in range(1, numPeers - 2):
            runTest(numPeers, failedPeers, repairDepth)

we need 3 extra functions 
uploadReplication: uploads file with replication factor X 
downloadReplicationCount: traverses the file tree and returns total found blocks
downloadRepairCount: traverses the file tree and returns total after repairing if needed
'''


peers_variations = [10,15,20]
fileSize = "25MB"

# results would be 
''' 

repair_depth -> {
    failed_peers -> [cnt0, cnt1, cnt2, .., cnt10]
}





'''
results = {
    "10" : {},
    "15" : {},
    "20" : {}
}

# read results from results.json
try:
    with open("results_reliability.json", "r") as f:
        results = json.load(f)
except:
    pass
    
depth = 5

for peers in peers_variations:
    for failed_peers in range(1, peers - 2, 2):
        # check how many runs we have done and subtract from 10
        total_runs = 10
        if str(failed_peers) in results[f"{peers}"]:
            total_runs -= len(results[f"{peers}"][str(failed_peers)])
        print(f"Num Peers: {peers}, Failed Peers: {failed_peers}, Total Runs: {total_runs}")
        for i in range(total_runs):   
            # Generate docker compose file
            create_test_env(peers, 0, 4, 0, 0, fileSize, community=False)
            
            # Wait for containers to start
            time.sleep(5)

            file_path = f"../test/data/largefile_{fileSize}.txt"
            upload_file(file_path, peers)
            
            file_cid = test_data[fileSize]["file_cid"]
            meta_cid = test_data[fileSize]["meta_cid"]
            while check_meta_pinned(meta_cid) == False:
                time.sleep(3)
                
            # give a chance for replication to take place
            time.sleep(120)
        

            # Kill failed peers
            kill_peers(failed_peers)

            # Download file
            
            count = download_count(file_cid, meta_cid, depth)
            print(f"Total blocks retrieved: {count}/{test_data[fileSize]['total_blocks']}")

            
            if str(failed_peers) not in results[f"{peers}"]:
                results[f"{peers}"][str(failed_peers)] = []
            
            results[f"{peers}"][str(failed_peers)].append(count)            

            #write results to results.json
            with open("results_reliability.json", "w") as f:
                json.dump(results, f, indent=4)
                
            stop_env()
