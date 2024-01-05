package Server

import (
	"github.com/gin-gonic/gin"
	"ipfs-alpha-entanglement-code/client"
	"sync"
	"time"
)

type IPConverter interface {
	ClusterToCommunityIP(clusterIP string) (communityIP string, err error)
}

type State struct {
	files                       map[string]*FileStats
	potentialFailedRegions      map[string][]string // map [region] -> [failed cluster peer names]
	running                     bool
	unavailableBlocksTimestamps []int64 // use UnixNano
}

type FileStats struct {
	fileCID                  string
	MetadataCID              string `json:"metadataCID"`
	StrandRootCID            string `json:"strandRootCID"`
	strandNumber             int
	DataBlocksMissing        map[uint]*WatchedBlock `json:"dataBlocksMissing,omitempty"`
	ParityBlocksMissing      map[uint]*WatchedBlock `json:"parityBlocksMissing,omitempty"`
	validParityBlocksHistory map[uint]*WatchedBlock // If too much mem used -> use a fifo/ring or make a gc
	EstimatedBlockProb       float32                `json:"estimatedBlockProb,omitempty"`
	Health                   float32                `json:"health,omitempty"`
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

type ForwardMonitoringRequest struct {
	FileCID        string   `json:"fileCID"`
	MetadataCID    string   `json:"metadataCID"`
	StrandRootCIDs []string `json:"strandRootCIDs"`
}

type StartMonitoringRequest struct {
	FileCID       string `json:"fileCID"`
	MetadataCID   string `json:"metadataCID"`
	StrandRootCID string `json:"strandRootCID"`
}

type StopMonitoringRequest struct {
	FileCID string `json:"fileCID"`
}

type ResetMonitoringRequest struct {
	FileCID string `json:"fileCID"`
	IsData  bool   `json:"isData"`
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
	FileCID                 string       `json:"fileCID"`
	MetaCID                 string       `json:"metaCID"`
	RepairStatus            map[int]bool `json:"repairStatus"`
	Origin                  string       `json:"origin"` // refers to the peer that was repairing these failures
	ParityAvailable         []bool       `json:"parityAvailable"`
	DataBlocksFetched       int          `json:"dataBlocksFetched"`
	DataBlocksCached        int          `json:"dataBlocksCached"`
	DataBlocksUnavailable   int          `json:"dataBlocksUnavailable"`
	DataBlocksError         int          `json:"dataBlocksError"`
	ParityBlocksFetched     int          `json:"parityBlocksFetched"`
	ParityBlocksCached      int          `json:"parityBlocksCached"`
	ParityBlocksUnavailable int          `json:"parityBlocksUnavailable"`
	ParityBlocksError       int          `json:"parityBlocksError"`
}

type UnitRepairOperation struct {
	FileCID       string
	MetaCID       string
	FailedIndices []int
	Depth         uint   // depth of repair in lattice
	Origin        string // refers to the target peer which is used when sending back result
}

type UnitRepairDone struct {
	FileCID                 string
	MetaCID                 string
	Origin                  string
	RepairStatus            map[int]bool
	ParityAvailable         []bool
	DataBlocksFetched       int
	DataBlocksCached        int
	DataBlocksUnavailable   int
	DataBlocksError         int
	ParityBlocksFetched     int
	ParityBlocksCached      int
	ParityBlocksUnavailable int
	ParityBlocksError       int
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

type DownloadMetrics struct {
	StartTime               *time.Time   `json:"startTime"`
	EndTime                 *time.Time   `json:"endTime"`
	Status                  RepairStatus `json:"status"`
	ParityAvailable         []bool       `json:"parityAvailable"`
	DataBlocksFetched       int          `json:"dataBlocksFetched"`
	DataBlocksCached        int          `json:"dataBlocksCached"`
	DataBlocksUnavailable   int          `json:"dataBlocksUnavailable"`
	DataBlocksError         int          `json:"dataBlocksError"`
	ParityBlocksFetched     int          `json:"parityBlocksFetched"`
	ParityBlocksCached      int          `json:"parityBlocksCached"`
	ParityBlocksUnavailable int          `json:"parityBlocksUnavailable"`
	ParityBlocksError       int          `json:"parityBlocksError"`
}

type CollabPeerInfo struct {
	Name                    string       `json:"name"`
	StartTime               time.Time    `json:"startTime"`
	EndTime                 time.Time    `json:"endTime"`
	Status                  RepairStatus `json:"status"`
	AllocatedBlocks         map[int]bool `json:"blocks"` // map from block CID to repair status
	ParityAvailable         []bool       `json:"parityAvailable"`
	DataBlocksFetched       int          `json:"dataBlocksFetched"`
	DataBlocksCached        int          `json:"dataBlocksCached"`
	DataBlocksUnavailable   int          `json:"dataBlocksUnavailable"`
	DataBlocksError         int          `json:"dataBlocksError"`
	ParityBlocksFetched     int          `json:"parityBlocksFetched"`
	ParityBlocksCached      int          `json:"parityBlocksCached"`
	ParityBlocksUnavailable int          `json:"parityBlocksUnavailable"`
	ParityBlocksError       int          `json:"parityBlocksError"`
}

type CollaborativeRepairData struct {
	FileCID   string                     `json:"fileCID"`
	MetaCID   string                     `json:"metaCID"`
	Depth     uint                       `json:"depth"`
	Status    RepairStatus               `json:"status"`
	StartTime time.Time                  `json:"startTime"`
	EndTime   time.Time                  `json:"endTime"`
	Peers     map[string]*CollabPeerInfo `json:"peers"` // from peer name to all associated information
	Origin    string                     `json:"origin"`

	// metrics used by the coordinating node
	ParityAvailable         []bool `json:"parityAvailable"`
	DataBlocksFetched       int    `json:"dataBlocksFetched"`
	DataBlocksCached        int    `json:"dataBlocksCached"`
	DataBlocksUnavailable   int    `json:"dataBlocksUnavailable"`
	DataBlocksError         int    `json:"dataBlocksError"`
	ParityBlocksFetched     int    `json:"parityBlocksFetched"`
	ParityBlocksCached      int    `json:"parityBlocksCached"`
	ParityBlocksUnavailable int    `json:"parityBlocksUnavailable"`
	ParityBlocksError       int    `json:"parityBlocksError"`
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

type CommunityNode struct {
	ClusterIP   string `json:"clusterIP"`
	ClusterPort int    `json:"cluserPort"` // Note the potential typo in 'cluserPort'
	IpfsIP      string `json:"ipfsIP"`
	IpfsPort    int    `json:"ipfsPort"`
}

type CommunityNodeAnnouncement struct {
	CommunityIP string `json:"communityIP"`
	ClusterIP   string `json:"clusterIP"`
	ClusterPort int    `json:"clusterPort"`
	IpfsIP      string `json:"ipfsIP"`
	IpfsPort    int    `json:"ipfsPort"`
}

type CommunitiesMap map[string]CommunityNode

type OperationType int

const (
	START_MONITOR_FILE OperationType = iota
	STOP_MONITOR_FILE
	RESET_MONITOR_FILE
)

type Operation struct {
	operationType OperationType
	parameter     []byte
}

type Server struct {
	ginEngine       *gin.Engine
	state           State
	stateMux        sync.Mutex
	operations      chan Operation
	ctx             chan struct{}
	client          *client.Client
	repairThreshold float32

	// data for collaborative repair
	// ipConverter IPConverter
	collabOps  chan *CollaborativeRepairOperation
	collabDone chan *CollaborativeRepairDone
	unitOps    chan *UnitRepairOperation
	unitDone   chan *UnitRepairDone
	strandOps  chan *StrandRepairOperation

	// data for stateful repair
	collabData map[string]*CollaborativeRepairData // map from [file CID] to repair data
	strandData map[string]*StrandRepairData        // map from [file CID + Strand] to repair data

	//personal information
	address          string //includes full address for community node include port
	clusterIP        string //includes only the IP/hostname of the cluster node
	clusterPort      int    //includes only the port of the cluster node
	ipfsIP           string //includes only the IP/hostname of the IPFS node
	ipfsPort         int    //includes only the port of the IPFS node
	discoveryAddress string //includes the full address of the discovery server
}
