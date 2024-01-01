from helpers import *
from test_data import test_data
import time 

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
        for failedPeers in range(1, numPeers - 2):
            for replicationFactor in [3, 5, 7]:
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


peers_variations = [5]
failed_peers_variations = [2]
repair_peers_variations = [3]
fileSizes = ["25MB"]
replication_factors = [4]
repair_depths = [5]

repair_depth = 5
replication_factor = 4
fileSize = "25MB"
repair_peers = 3

for peers in peers_variations:
    for failed_peers in failed_peers_variations:
        
        # Generate docker compose file
        create_test_env(peers, repair_depth, replication_factor, failed_peers, repair_peers, fileSize)
        
        # Wait for containers to start
        time.sleep(20)

        file_path = f"../test/data/largefile_{fileSize}.txt"
        upload_file(file_path, replication_factor)

        # TODO: add timeout
        # Wait for metadata to be pinned
        # TODO: make sure that all parities have been pinned first
        meta_cid = test_data[fileSize]["meta_cid"]
        while check_meta_pinned(meta_cid) == False:
            time.sleep(3)

        # TODO: maybe wait before killing 
        time.sleep(20)

        # Kill failed peers
        kill_peers(failed_peers)

        # TODO: add timeout
        # TODO: something is wrong here, takes too long
        # wait until discovery is updated
        # while check_community_peers() != peers - failed_peers:
        #     print("waiting for discovery to update")
        #     time.sleep(3)
        
        # Download file
        file_cid = test_data[fileSize]["file_cid"]
        count = download_count(file_cid, meta_cid, repair_depth)
        print(f"Total blocks retrieved: {count}/{test_data[fileSize]['total_blocks']}")


        stop_env()
            