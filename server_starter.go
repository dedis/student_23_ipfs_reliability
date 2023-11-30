package main

import "ipfs-alpha-entanglement-code/Server"

func community_node_start() {
	server := &Server.Server{}
	server.RunServer()
}
