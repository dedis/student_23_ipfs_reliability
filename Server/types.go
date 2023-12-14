package Server

import (
	"ipfs-alpha-entanglement-code/client"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type IPConverter interface {
	ClusterToCommunityIP(clusterIP string) (communityIP string, err error)
}

type State struct {
	files map[string]FileStats
	// TODO: Add more state about the cluster?
	//  time since last expired ping, avg time to failure
	potentialFailedRegions map[string][]string // map [region] -> [failed cluster peer names]
}

type FileStats struct {
	StrandRootCID       string                `json:"strandRootCID"`
	DataBlocksMissing   map[uint]WatchedBlock `json:"dataBlocksMissing,omitempty"`
	ParityBlocksMissing map[uint]WatchedBlock `json:"parityBlocksMissing,omitempty"`
	EstimatedBlockProb  float32               `json:"estimatedBlockProb,omitempty"`
	Health              float32               `json:"health,omitempty"`
}

type WatchedBlock struct {
	CID         string      `json:"blockCID"`
	Peer        ClusterPeer `json:"hostPeer"`
	Probability float32     `json:"prob"` // Presence probability (account for transient failures)
}

type ClusterPeer struct {
	Name   string `json:"peerName"`
	Region string `json:"region"`
}

type CollaborativeRepairOperation struct {
	FileCID  string
	MetaCID  string
	Depth    uint   // depth of repair in lattice
	Origin   string // refers to original requester to send back the result
	NumPeers int    // number of peers to use for repair
}

type CollaborativeRepairDone struct {
	FileCID      string
	MetaCID      string
	Origin       string // refers to the peer that was repairing these failures
	RepairStatus bool
}

type CollaborativeRepairOperationRequest struct {
	FileCID  string `json:"fileCID"`
	MetaCID  string `json:"metaCID"`
	Depth    uint   `json:"depth"`    // depth of repair in lattice
	Origin   string `json:"origin"`   // refers to the target peer which is used when sending back result
	NumPeers int    `json:"numPeers"` // number of peers to use for repair
}

// This response is async, it is sent back to the origin of the request when the repair is done
type CollaborativeRepairOperationResponse struct {
	FileCID      string `json:"fileCID"`
	MetaCID      string `json:"metaCID"`
	RepairStatus bool   `json:"repairStatus"`
	Origin       string `json:"origin"` // refers to the peer that was repairing these failures
}

type UnitRepairOperationRequest struct {
	FileCID       string `json:"fileCID"`
	MetaCID       string `json:"metaCID"`
	FailedIndices []int  `json:"failedIndices"`
	Depth         uint   `json:"depth"`
	Origin        string `json:"origin"` // refers to the target peer which is used when sending back result
}

// This response is async, it is sent back to the origin of the request when the repair is done
type UnitRepairOperationResponse struct {
	FileCID      string       `json:"fileCID"`
	MetaCID      string       `json:"metaCID"`
	RepairStatus map[int]bool `json:"repairStatus"`
	Origin       string       `json:"origin"` // refers to the peer that was repairing these failures
}

type UnitRepairOperation struct {
	FileCID       string
	MetaCID       string
	FailedIndices []int
	Depth         uint   // depth of repair in lattice
	Origin        string // refers to the target peer which is used when sending back result
}

type UnitRepairDone struct {
	FileCID      string
	MetaCID      string
	Origin       string
	RepairStatus map[int]bool
}

type StrandRepairOperation struct {
	FileCID string
	MetaCID string
	Strand  int
	Depth   uint
}

type StrandRepairOperationRequest struct {
	FileCID string `json:"fileCID"`
	MetaCID string `json:"metaCID"`
	Strand  int    `json:"strand"`
	Depth   uint   `json:"depth"`
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
	Status          RepairStatus
	AllocatedBlocks map[int]bool // map from block CID to repair status
}

type CollaborativeRepairData struct {
	FileCID   string
	MetaCID   string
	Depth     uint
	Status    RepairStatus
	StartTime time.Time
	EndTime   time.Time
	Peers     map[string]*CollabPeerInfo // from peer name to all associated information
	Origin    string
}

type StrandRepairData struct {
	FileCID   string
	MetaCID   string
	Strand    int
	Depth     uint
	Status    RepairStatus
	StartTime time.Time
	EndTime   time.Time
}

type OperationType int

const (
	START_MONITOR_FILE OperationType = iota
	STOP_MONITOR_FILE
)

type Operation struct {
	operationType OperationType
	parameter     string
	// TODO: Define more params or custom fields?
}

type Server struct {
	ginEngine       *gin.Engine
	state           State
	stateMux        sync.Mutex
	operations      chan Operation
	ctx             chan struct{}
	client          *client.Client
	address         string
	repairThreshold float32

	// data for collaborative repair
	ipConverter IPConverter
	collabOps   chan *CollaborativeRepairOperation
	collabDone  chan *CollaborativeRepairDone
	unitOps     chan *UnitRepairOperation
	unitDone    chan *UnitRepairDone
	strandOps   chan *StrandRepairOperation

	// data for stateful repair
	collabData map[string]*CollaborativeRepairData // map from [file CID] to repair data
	strandData map[string]*StrandRepairData        // map from [file CID + Strand] to repair data
}
