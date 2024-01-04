# Helper functions for running tests
import jinja2
import os 
import time
import requests
import uuid
import json

def generate_docker_compose(N, depth, replication_factor, failed_peers, repair_peers, file_size, community=True):
    # Load Jinja2 template
    templateLoader = jinja2.FileSystemLoader(searchpath="./")
    templateEnv = jinja2.Environment(loader=templateLoader)
    template = templateEnv.get_template("docker-compose-template.j2")

    # Render the template with the given number of peers
    output = template.render(N=N, DEPTH=depth, REPLICATION_FACTOR=replication_factor, FAILED_PEERS=failed_peers, REPAIR_PEERS=repair_peers, FILE_SIZE=file_size, community=community)

    # Write the output to a file
    with open("docker-compose.yaml", "w") as f:
        f.write(output)

# Creates a test environment with the given parameters and starts the containers
def create_test_env(num_peers, repair_depth, replication_factor, failed_peers, repair_peers, file_size, community=True):
    # Generate docker compose file
    generate_docker_compose(num_peers, repair_depth, replication_factor, failed_peers, repair_peers, file_size, community)

    # Start the containers
    os.system("docker-compose up -d")

    # Wait for containers to start
    time.sleep(5)

# Uploads file to ipfs0 
def upload_file(file_path, replication_factor):
    # Upload the file
    abs_file_path = os.path.abspath(file_path)
    abs_exec_path = os.path.abspath("../main.go")

    os.system(f"go run {abs_exec_path} upload {abs_file_path} --alpha 3 -s 5 -p 5 -r {str(replication_factor)}")

# Uploads file to ipfs0 with direct replication
def upload_file_direct_replication(file_path, replication_factor):
    # Upload the file
    abs_file_path = os.path.abspath(file_path)
    abs_exec_path = os.path.abspath("../main.go")

    os.system(f"go run {abs_exec_path} upload {abs_file_path} -t {replication_factor}")


def kill_peers(num_peers):
    print(f"Killing {num_peers} peers")
    # kill num_peer containers except index 1
    os.system(f"docker kill ipfs0  cluster0")

    for i in range(2, num_peers+1):
        os.system(f"docker kill ipfs{i} cluster{i}")


def kill_peers_community(num_peers):
    print(f"Killing {num_peers} peers")
    # kill num_peer containers except index 1
    os.system(f"docker kill ipfs0 community0 cluster0")

    for i in range(2, num_peers+1):
        os.system(f"docker kill ipfs{i} community{i} cluster{i}")

def kill_peers_range(start, num_peers):
    if start == 0:
        print("Killing IPFS0")
        os.system(f"docker kill ipfs0 community0 cluster0")
        start = 2
    
    for i in range(start, num_peers+1):
        print(f"Killing IPFS{i}")
        os.system(f"docker kill ipfs{i} community{i} cluster{i}")
    
    return num_peers+1

def stop_env():
    os.system("docker-compose down")
    os.system("docker volume prune -f")
    os.system("docker-compose rm -f")

def create_tmp_file():
    # Create tmp file to store output
    tmp_file = open("tmp.txt", "w")
    tmp_file.close()

def download_count(file_cid, meta_cid, repair_depth, metrics=False):
    abs_exec_path = os.path.abspath("../main.go")
    tmp_file_path = os.path.abspath(f"{uuid.uuid4()}.txt")
    metrics_flag = ""
    if metrics:
        metrics_flag = "-t"

    if meta_cid == "":
        os.system(f"go run {abs_exec_path} downloadcnt {file_cid} -o {tmp_file_path} -d {repair_depth}")
    else:
        os.system(f"go run {abs_exec_path} downloadcnt {file_cid} -m {meta_cid} -o {tmp_file_path} -d {repair_depth} {metrics_flag}")
    
    if metrics:
        try:
            # Read metrics from file
            with open(tmp_file_path, "r") as f:
                metrics = json.load(f)
            
            os.remove(tmp_file_path)
        except Exception as e:
            print(e)
            metrics = {}
        return metrics

    try:   
        with open(tmp_file_path, "r") as f:
            count = int(f.read())
        
        os.remove(tmp_file_path)
    except Exception as e:
        print(e)
        count = 0
        
    return count

def single_download(file_cid, meta_cid, repair_depth):
    # Always download from ipfs1 since we assume ipfs0 (uploader) is down
    abs_exec_path = os.path.abspath("../main.go")
    os.system(f"go run {abs_exec_path} download {file_cid} -m {meta_cid}  -o out -d {repair_depth} -a localhost:7071")

def collab_repair(file_cid, meta_cid, repair_depth, repair_peers):
    os.system(f"curl -X POST -H 'Content-Type: application/json' -d '{{\"fileCID\" : \"{file_cid}\", \"metaCID\" : \"{meta_cid}\", \"depth\" : {repair_depth}, \"origin\" : \"\", \"numPeers\" : {repair_peers}}}' http://localhost:7071/triggerCollabRepair")

def check_done():
    # Check if all peers are done
    print("Checking if all peers are done")
    try:
        # Send a GET request to the specified URL
        response = requests.get("http://localhost:3000/done")
        # print(response.json())
        # Check if the request was successful
        if response.status_code == 200:
            # Assuming the response is a boolean (True or False)
            return response.json() == True
        else:
            print("Failed to get data. Status code:", response.status_code)
            return False
    except Exception as e:
        print("An error occurred:", e)
        return False

def retrieve_metrics():
    print("Retrieving metrics")
    try:
        response = requests.get("http://localhost:3000/metrics")
        if response.status_code == 200:
            return response.json()
        else:
            print("Failed to get data. Status code:", response.status_code)
            return {}
    except Exception as e:
        print("An error occurred:", e)
        return {}

def check_meta_pinned(meta_cid):
    # Get the pin status from cluster1 in case cluster0 is already down
    # Shouldn't be the case since we should always wait until metadata is pinned before killing ipfs0
    print("Checking if metadata is pinned")
    response = requests.get(f"http://localhost:9095/pins/{meta_cid}")
    # print(response.json())
    # Check if the request was successful
    if response.status_code == 200:
        data = response.json()

        # Check if all peers have the status 'pinned'
        for peer_id, peer_info in data.get("peer_map", {}).items():
            if peer_info.get("status") != "pinned":
                return False

        # If all peers have status 'pinned'
        return True
    else:
        # If the request failed
        return False

def check_community_peers():
    print("Checking number of peers in community")
    try:
        # Send a GET request to the specified URL
        response = requests.get(f"http://localhost:3000/peers")
        # print(response.json())
        # Check if the request was successful
        if response.status_code == 200:
            data = response.json()

            # Count the number of peers
            number_of_peers = len(data)
            return number_of_peers
        else:
            print("Failed to get data. Status code:", response.status_code)
            return None
    except Exception as e:
        print("An error occurred:", e)
        return None