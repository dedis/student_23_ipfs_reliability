package integration

import (
	"ipfs-alpha-entanglement-code/Server"
	"testing"
)

// Manual testing for now
func Test_API(t *testing.T) {
	server := &Server.Server{}
	server.RunServer(8080)
}
