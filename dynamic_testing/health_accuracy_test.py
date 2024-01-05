from helpers import *
from test_data import test_data
import time
import json

# 10 test files (each slightly different to get different CIDs)
file = "community_test_18MB_1.txt"
file_cid = "Qmb9Qewfdst9rEm5eFV7JNaUrKtw6kPgMcQ83154sfnMa2"
file_meta_cid = "QmVQyNEsvPVHVNA777eLDa4onRbshgcUzCEFq5QwWiVRQw"

peers = 10
failed_peers = 2
number_health_check = 5
# Generate docker compose file
create_test_env(peers, 5, 5, 0, 0, "25MB")

# Wait for containers to start
time.sleep(5)

# Upload the file
file_path = f"../test/data/community/{file}"
upload_file_community(file_path, peers)
time.sleep(5)

# Kill peers
kill_peers_community(failed_peers)

# Wait for discovery to update
time.sleep(3)

try:
    with open("output_health_scores.txt", "a") as f:
        for i in range(number_health_check):
            f.write(f"Test{i}:\n")
            ret = compute_community_file_health(file_cid)
            if ret != None:
                print(f"Health calc ok")
                print(ret)
                f.write(f"Calc[{i}] with Failed peers={failed_peers}")
                f.write(ret["Result"])
            else:
                print(f"Checked health failed at test {i}")
                f.write(f"Checked health failed at test {i}")

            if i != number_health_check - 1:
                print(f"Sleeping for 5 seconds")
                time.sleep(5)

        # Check if could indeed repair the file with a depth of 5
        print("Try to download file")
        single_download(file_cid, file_meta_cid, 5)
        # Manually check if the file was downloaded

except Exception as e:
    print(e)
    pass


print("Stopping env")
stop_env()
