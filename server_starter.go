package main

import "ipfs-alpha-entanglement-code/Server"

func community_node_start(port int) {
	server := &Server.Server{}
	server.RunServer(port)
}
