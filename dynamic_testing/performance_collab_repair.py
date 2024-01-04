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

repair_peers = 3
repair_depths = [10]
fileSize = "25MB"

# results would be 
''' 

repair_depth -> {
    failed_peers -> [result1, result2, result3,..]
}

'''

results = {
    "depth_5": {},
    "depth_7": {},
    "depth_10": {}
}

# read results from results.json
try:
    with open("results_performance_collab_repair.json", "r") as f:
        results = json.load(f)
except:
    pass
    
peers = 20

for depth in repair_depths:
    for failed_peers in range(1, 10, 2):
        # check how many runs we have done and subtract from 10
        total_runs = 10
        if str(failed_peers) in results[f"depth_{depth}"]:
            total_runs -= len(results[f"depth_{depth}"][str(failed_peers)])
        print(f"Depth: {depth}, Failed Peers: {failed_peers}, Total Runs: {total_runs}")
        for i in range(total_runs):
            stop_env()   
            # Generate docker compose file
            create_test_env(peers, 0, 4, 0, 0, fileSize, community=True)
            
            # Wait for containers to start
            time.sleep(5)

            file_path = f"../test/data/largefile_{fileSize}.txt"
            upload_file(file_path, peers)
            
            file_cid = test_data[fileSize]["file_cid"]
            meta_cid = test_data[fileSize]["meta_cid"]

            startTime = time.time()

            # Wait for file to be pinned or timeout (5 mins)
            while check_meta_pinned(meta_cid) == False and time.time() - startTime < 300:
                time.sleep(3)
                
            if time.time() - startTime >= 300:
                print("File upload timed out")
                continue

            # give a chance for replication to take place
            time.sleep(120)
        

            # Kill failed peers
            kill_peers_community(failed_peers)


            # Wait for discovery to update or timeout (5 mins)
            startTime = time.time()
            while (check_community_peers() != peers - failed_peers) and (time.time() - startTime < 300):
                print("waiting for discovery to update")
                time.sleep(3)

            if time.time() - startTime >= 300:
                print("Discovery timed out")
                continue

            # Trigger collab repair
            collab_repair(file_cid, meta_cid, repair_depth=depth, repair_peers=repair_peers)

            startTime = time.time()
            # Wait for repair to finish or timeout (30 mins)
            while check_done() == False and time.time() - startTime < 1800:
                print("waiting for repair to finish")
                time.sleep(3)
            
            if time.time() - startTime >= 1800:
                print("Repair timed out")
                continue

            metrics = retrieve_metrics()

            
            if str(failed_peers) not in results[f"depth_{depth}"]:
                results[f"depth_{depth}"][str(failed_peers)] = []
            
            results[f"depth_{depth}"][str(failed_peers)].append(metrics)            

            #write results to results.json
            with open("results_performance_collab_repair.json", "w") as f:
                json.dump(results, f, indent=4)
                
            stop_env()
