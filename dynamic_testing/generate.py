import argparse
import helpers



def main():
    # Create the parser
    parser = argparse.ArgumentParser(description='Generate Docker Compose file for IPFS clusters.')

    # Add the arguments
    parser.add_argument('-N', type=int, help='Number of IPFS,cluster,community nodes to create')
    parser.add_argument('--depth', '-d' , type=int, default=5 ,help='Depth of repair in the lattice')
    parser.add_argument('--replication_factor', '-p', type=int, default=4, help='Replication factor of the lattice')
    parser.add_argument('--failed_peers', '-f', type=int, default=2, help='Number of failed peers')
    parser.add_argument('--repair_peers', '-r', type=int, default=3, help='Number of repair peers')
    parser.add_argument('--file_size', '-s', type=str, default="25MB", help='Size of file to upload in bytes')                    

    # Execute the parse_args() method
    args = parser.parse_args()

    # Generate docker compose file
    helpers.generate_docker_compose(args.N, args.depth, args.replication_factor, args.failed_peers, args.repair_peers, args.file_size)

if __name__ == "__main__":
    main()
