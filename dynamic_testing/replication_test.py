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


peers_variations = [10,15,20,25,30]
replication_factors = [3,5,7]
fileSize = "25MB"

# results would be 
''' 

num_peers -> {
    replication_factor -> {
        failed_peers -> [cnt0, cnt1, cnt2, .., cnt10]
    }
}




'''
results = {
    "10" : {
        "replication3" : {},
        "replication5" : {},
        "replication7" : {},
    },
    "15" : {
        "replication3" : {},
        "replication5" : {},
        "replication7" : {},
    },
    "20" : {
        "replication3" : {},
        "replication5" : {},
        "replication7" : {},
    },
    "25" : {
        "replication3" : {},
        "replication5" : {},
        "replication7" : {},
    },
    "30" : {
        "replication3" : {},
        "replication5" : {},
        "replication7" : {},
    },
}


for peers in peers_variations:
    for failed_peers in range(1, peers - 2, 2):
        for replication_factor in replication_factors:
            for i in range(10):
                # Generate docker compose file
                create_test_env(peers, 0, replication_factor, failed_peers, 0, fileSize)
                
                # Wait for containers to start
                time.sleep(10)

                file_path = f"../test/data/largefile_{fileSize}.txt"
                upload_file_direct_replication(file_path, replication_factor)


                # give a chance for replication to take place
                time.sleep(20)

                # Kill failed peers
                kill_peers(failed_peers)

                # Download file
                file_cid = test_data[fileSize]["file_cid"]
                count = download_count(file_cid, "", 1)
                print(f"Total blocks retrieved: {count}/{test_data[fileSize]['total_blocks']}")

                stop_env()

                if str(failed_peers) not in results[str(peers)][f"replication{replication_factor}"]:
                    results[str(peers)][f"replication{replication_factor}"][str(failed_peers)] = []

                results[str(peers)][f"replication{replication_factor}"][str(failed_peers)].append(count)

                #write results to results.json
                with open("results.json", "w") as f:
                    json.dump(results, f, indent=4)
