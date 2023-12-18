import jinja2
import argparse

def generate_docker_compose(N, depth, replication_factor, failed_peers, repair_peers):
    # Load Jinja2 template
    templateLoader = jinja2.FileSystemLoader(searchpath="./")
    templateEnv = jinja2.Environment(loader=templateLoader)
    template = templateEnv.get_template("docker-compose-template.j2")

    # Render the template with the given number of peers
    output = template.render(N=N, DEPTH=depth, REPLICATION_FACTOR=replication_factor, FAILED_PEERS=failed_peers, REPAIR_PEERS=repair_peers)

    # Write the output to a file
    with open("docker-compose.yaml", "w") as f:
        f.write(output)


def main():
    # Create the parser
    parser = argparse.ArgumentParser(description='Generate Docker Compose file for IPFS clusters.')

    # Add the arguments
    parser.add_argument('-N', type=int, help='Number of IPFS,cluster,community nodes to create')
    parser.add_argument('--depth', '-d' , type=int, default=5 ,help='Depth of repair in the lattice')
    parser.add_argument('--replication_factor', '-p', type=int, default=4, help='Replication factor of the lattice')
    parser.add_argument('--failed_peers', '-f', type=int, default=2, help='Number of failed peers')
    parser.add_argument('--repair_peers', '-r', type=int, default=3, help='Number of repair peers')
                        

    # Execute the parse_args() method
    args = parser.parse_args()

    # Generate docker compose file
    generate_docker_compose(args.N, args.depth, args.replication_factor, args.failed_peers, args.repair_peers)

if __name__ == "__main__":
    main()
