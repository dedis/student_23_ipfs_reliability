package Server

import (
	"ipfs-alpha-entanglement-code/client"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
)

type IPConverter interface {
	ClusterToCommunityIP(clusterIP string) (communityIP string, err error)
}

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

type CollaborativeRepairOperation struct {
	FileCID string
	MetaCID string
	Depth   int    // depth of repair in lattice
	Origin  string // refers to original requester to send back the result
	Peer    string // refers to the target peer which is used when sending back result
}

type CollaborativeRepairDone struct {
	FileCID      string
	MetaCID      string
	Peer         string // refers to the peer that was repairing these failures
	RepairStatus bool
}

type CollaborativeRepairOperationRequest struct {
	FileCID string `json:"fileCID"`
	MetaCID string `json:"metaCID"`
	Depth   int    `json:"depth"` // depth of repair in lattice
	Peer    string `json:"peer"`  // refers to the target peer which is used when sending back result
}

// This response is async, it is sent back to the origin of the request when the repair is done
type CollaborativeRepairOperationResponse struct {
	FileCID      string `json:"fileCID"`
	MetaCID      string `json:"metaCID"`
	RepairStatus bool   `json:"repairStatus"`
	Peer         string `json:"peer"` // refers to the peer that was repairing these failures
}

type UnitRepairOperationRequest struct {
	FileCID    string   `json:"fileCID"`
	MetaCID    string   `json:"metaCID"`
	FailedCIDs []string `json:"failedCIDs"`
	Depth      int      `json:"depth"`
	Peer       string   `json:"peer"` // refers to the target peer which is used when sending back result
}

// This response is async, it is sent back to the origin of the request when the repair is done
type UnitRepairOperationResponse struct {
	FileCID      string          `json:"fileCID"`
	MetaCID      string          `json:"metaCID"`
	RepairStatus map[string]bool `json:"repairStatus"`
	Peer         string          `json:"peer"` // refers to the peer that was repairing these failures
}

type UnitRepairOperation struct {
	FileCID    string
	MetaCID    string
	FailedCIDs []string
	Origin     string // refers to original requester to send back the result
	Depth      int    // depth of repair in lattice
	Peer       string // refers to the target peer which is used when sending back result
}

type UnitRepairDone struct {
	FileCID      string
	MetaCID      string
	Peer         string
	RepairStatus map[string]bool
}

type StrandRepairOperation struct {
	FileCID string
	MetaCID string
	Strand  int
}

type StrandRepairOperationRequest struct {
	FileCID string `json:"fileCID"`
	MetaCID string `json:"metaCID"`
	Strand  int    `json:"strand"`
}

type RepairStatus int

const (
	PENDING RepairStatus = iota
	SUCCESS
	FAILURE
)

type CollabPeerInfo struct {
	Name            string
	StartTime       time.Time
	EndTime         time.Time
	AllocatedBlocks map[string]RepairStatus // map from block CID to repair status
}
type CollaborativeRepairData struct {
	FileCID   string
	MetaCID   string
	Status    RepairStatus
	StartTime time.Time
	EndTime   time.Time
	Peers     map[string]CollabPeerInfo // from peer name to all associated information
}

type StrandRepairData struct {
	FileCID   string
	MetaCID   string
	Strand    int
	Status    RepairStatus
	StartTime time.Time
	EndTime   time.Time
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

	// data for collaborative repair
	ipConverter IPConverter
	collabOps   chan CollaborativeRepairOperation
	collabDone  chan CollaborativeRepairDone
	unitOps     chan UnitRepairOperation
	unitDone    chan UnitRepairDone
	strandOps   chan StrandRepairOperation

	// data for stateful repair
	collabData map[string]CollaborativeRepairData // map from [file CID] to repair data
	strandData map[string]StrandRepairData        // map from [file CID + Strand] to repair data
}
