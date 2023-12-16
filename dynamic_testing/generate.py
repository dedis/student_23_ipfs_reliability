import jinja2
import argparse

def generate_docker_compose(N):
    # Load Jinja2 template
    templateLoader = jinja2.FileSystemLoader(searchpath="./")
    templateEnv = jinja2.Environment(loader=templateLoader)
    template = templateEnv.get_template("docker-compose-template.j2")

    # Render the template with the given number of peers
    output = template.render(N=N)

    # Write the output to a file
    with open("docker-compose.yaml", "w") as f:
        f.write(output)


def main():
    # Create the parser
    parser = argparse.ArgumentParser(description='Generate Docker Compose file for IPFS clusters.')

    # Add the arguments
    parser.add_argument('N', type=int, help='Number of IPFS,cluster,community nodes to create')

    # Execute the parse_args() method
    args = parser.parse_args()

    # Generate docker compose file
    generate_docker_compose(args.N)

if __name__ == "__main__":
    main()
