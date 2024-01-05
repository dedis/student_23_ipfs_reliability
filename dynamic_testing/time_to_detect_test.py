from helpers import *
from test_data import test_data
import time
import json

# 10 test files (each slightly different to get different CIDs)
files = ["community_test_18MB_1.txt", "community_test_18MB_2.txt", "community_test_18MB_3.txt", "community_test_18MB_4.txt", "community_test_18MB_5.txt", "community_test_18MB_6.txt", "community_test_18MB_7.txt", "community_test_18MB_8.txt", "community_test_18MB_9.txt", "community_test_18MB_10.txt"]
files_cid = ["Qmb9Qewfdst9rEm5eFV7JNaUrKtw6kPgMcQ83154sfnMa2", "QmeB17S96ywumRD1o9R42tEvmSai9UUQTpLQ9vRuQbgMGK", "QmWkDoKXkHr7C721QLmkvvXZbeyx4PAq153PmxgS7iCkgM", "QmbpYaZ5sc2wSUY1J1JKG16EotdkwXZgBM2ginqiTnwTbC","QmZ3kkUVN9JkFx8VYzmh5R18h9ahBLtcGEYYJYPDZLniha","QmSyqt6QMSDAQzLn3MmBcLEPHZQJs1FgQ75WDzsZA1CqZs","QmWM5AYPgH3NvHqueJbedN3uDuNKBnnnjXg1w7Gnh2J67s","QmRqJewUSYx5sFE8TVR1cawP7gNpcv78G3F88eCH2N8ZFM","QmPBkEwY6rHAE2P9vLFPMjFR2idV4zrzP6qcR3kcTY6xSZ","QmPwsxs7eEBxmPHaxjdUBWy958sJ5CX26L6hyy3QfPYEyx"]

peers = 4
number_tests = 10 # steps of 120 seconds after which the block prob is measured

# Generate docker compose file
create_test_env(peers, 2, 3, 0, 0, "25MB")

# Wait for containers to start
time.sleep(5)

# Upload the files
for file in files:
    file_path = f"../test/data/community/{file}"
    upload_file_community(file_path, peers)
    time.sleep(10)

time.sleep(15)

# Kill peers 0 and 2, all data blocks and some parities are gone
kill_peers_community(2)

try:
    with open("output_monitoring.txt", "w") as f:
        for i in range(number_tests):
            print(f"Test {i}")
            f.write(f"Test{i}:\n")
            for cid in files_cid:
                ret = check_community_file_status(cid)
                if ret != None:
                    print(f"Checked cid ok")
                    print(ret)
                    f.write(ret["Result"])
                else:
                    print(f"Checked cid[{cid}] failed at test {i}")
                    f.write(f"Checked cid[{cid}] failed at test {i}")

            if i != number_tests - 1:
                print(f"Sleeping for 120 seconds")
                time.sleep(120)
except Exception as e:
    print(e)
    pass


print("Stopping environment in 1 minute")
time.sleep(60)
stop_env()
