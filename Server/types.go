package Server

import (
	"ipfs-alpha-entanglement-code/client"
	"sync"

	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
)

type State struct {
	files map[string]FileStats
	// TODO: Add state about the cluster
	//  failed peers, time since last expired ping, avg time to failure
}

type FileStats struct {
	strandCID           string
	numBlocks           int
	dataBlocksMissing   map[uint]WatchedBlock
	parityBlocksMissing map[uint]WatchedBlock
	estimatedBlockProb  float32
	health              float32
}

type WatchedBlock struct {
	CID         string
	peer        ClusterPeer
	probability float32 // Presence probability (account for transient failures)
}

type ClusterPeer struct {
	CID    string
	region string
}

type OperationType int

const (
	START_MONITOR_FILE OperationType = iota
	STOP_MONITOR_FILE
	REPARE_FILE
)

type Operation struct {
	operationType OperationType
	parameter     string
	// TODO: Define more params or custom fields?
}

type Server struct {
	ginEngine  *gin.Engine
	sh         *shell.Shell
	state      State
	stateMux   sync.Mutex
	operations chan Operation
	ctx        chan struct{}
	client     *client.Client
}
