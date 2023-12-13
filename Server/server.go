package Server

import (
	"fmt"
	"ipfs-alpha-entanglement-code/client"
	"ipfs-alpha-entanglement-code/docker"
	"ipfs-alpha-entanglement-code/util"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
)

// SetUpServer
// @Description: Initialize the Server struct and the REST endpoints
func (s *Server) setUpServer() {
	s.ginEngine = gin.Default()

	// monitoring endpoints
	s.ginEngine.POST("/startMonitorFile", func(c *gin.Context) { startMonitorFile(s, c) })
	s.ginEngine.POST("/stopMonitorFile", func(c *gin.Context) { stopMonitorFile(s, c) })
	s.ginEngine.GET("/listMonitor", func(c *gin.Context) { listMonitor(s, c) })
	s.ginEngine.GET("/checkFileStatus", func(c *gin.Context) { checkFileStatus(s, c) })
	s.ginEngine.GET("/checkClusterStatus", func(c *gin.Context) { checkClusterStatus(s, c) })

	// repair endpoints
	s.ginEngine.GET("/downloadFile", func(c *gin.Context) { downloadFile(s, c) })
	s.ginEngine.POST("/triggerCollabRepair", func(c *gin.Context) { triggerCollabRepair(s, c) })
	s.ginEngine.POST("/triggerUnitRepair", func(c *gin.Context) { triggerUnitRepair(s, c) })
	s.ginEngine.POST("/triggerStrandRepair", func(c *gin.Context) { triggerStrandRepair(s, c) })
	s.ginEngine.POST("/reportUnitRepair", func(c *gin.Context) { reportUnitRepair(s, c) })
	s.ginEngine.POST("/reportCollabRepair", func(c *gin.Context) { reportCollabRepair(s, c) })

	// init state
	s.ctx = make(chan struct{})
	s.operations = make(chan Operation)
	s.state = State{files: make(map[string]FileStats)}
	s.sh = shell.NewLocalShell() // FIXME need to call NewShell?
	s.client = &client.Client{}
	s.ipConverter = &docker.DockerClusterToCommunityConverter{}
	s.collabOps = make(chan *CollaborativeRepairOperation)
	s.collabDone = make(chan *CollaborativeRepairDone)
	s.unitOps = make(chan *UnitRepairOperation)
	s.unitDone = make(chan *UnitRepairDone)
	s.strandOps = make(chan *StrandRepairOperation)
	s.collabData = make(map[string]*CollaborativeRepairData)
	s.strandData = make(map[string]*StrandRepairData)
}

// RunServer
// @Description: Run the server (blocking)
// @param port: The port to listen on
func (s *Server) RunServer(port int) int {
	s.setUpServer()

	// TODO: Update address to reflect actual address seen by others
	s.address = fmt.Sprintf("localhost:%d", port)
	util.LogPrintf("Server listening on %s", s.address)

	// Starting daemon
	go Daemon(s)
	defer close(s.ctx)

	err := s.ginEngine.Run(fmt.Sprintf(":%d", port)) // blocking
	// listen and serve on 0.0.0.0 + port
	if err != nil {
		return 1
	}

	return 0
}

// startMonitorFile
// Query parameters in context: dataRoot (CID-string), strandParityRoot (CID-string), numBlocks (int)
func startMonitorFile(s *Server, c *gin.Context) {
	dataRoot := c.Query("dataRoot")
	strandParityRoot := c.Query("strandParityRoot")
	numBlocks := c.Query("numBlocks")

	if dataRoot == "" || strandParityRoot == "" || numBlocks == "" {
		c.JSON(400, gin.H{"message": "Missing CID parameters"})
		return
	}

	params := []string{dataRoot, strandParityRoot, numBlocks}

	s.operations <- Operation{START_MONITOR_FILE, strings.Join(params, ",")}

	c.JSON(200, gin.H{"message": "Start op."})
}

// stopMonitorFile
// Query parameters in context: dataRoot (CID)
func stopMonitorFile(s *Server, c *gin.Context) {
	dataRoot := c.Query("dataRoot")

	if dataRoot == "" {
		c.JSON(400, gin.H{"message": "Missing CID parameter"})
		return
	}

	s.operations <- Operation{STOP_MONITOR_FILE, dataRoot}

	c.JSON(200, gin.H{"message": "Stop op."})
}

func listMonitor(s *Server, c *gin.Context) {
	s.stateMux.Lock()
	defer s.stateMux.Unlock()

	// gather CIDs of files being monitored
	cids := ""

	if len(s.state.files) == 0 {
		c.JSON(200, gin.H{"message": "No monitored CIDs"})
		return
	}

	for file, stats := range s.state.files {
		cids += "CID=" + file + "-[Health = " +
			strconv.FormatFloat(float64(stats.Health()), 'f', -1, 64) + "%], "
	}

	c.JSON(200, gin.H{"message": "Listing monitored CIDs", "CIDs": cids[:len(cids)-2]})
}

func checkFileStatus(s *Server, c *gin.Context) {
	// TODO implement
	c.JSON(503, gin.H{"message": "Not Yet implemented"})
}

func checkClusterStatus(s *Server, c *gin.Context) {
	// TODO implement
	c.JSON(503, gin.H{"message": "Not Yet implemented"})
}

// Query parameters in context: rootFileCID (CID-string), metadataCID (CID-string), path (string), uploadRecoverData (bool)
func downloadFile(s *Server, c *gin.Context) {
	rootFileCID := c.Query("rootFileCID")
	metadataCID := c.Query("metadataCID")
	path := c.Query("path")
	uploadRecoverData := c.Query("uploadRecoverData")

	options := client.DownloadOption{
		UploadRecoverData: uploadRecoverData == "true",
		MetaCID:           metadataCID,
	}

	out, err := s.client.Download(rootFileCID, path, options)
	if err != nil {
		c.JSON(400, gin.H{"message": "Download failed", "error": err.Error()})
		return
	} else {
		c.JSON(200, gin.H{"message": "Downloaded", "out": out})
	}
}

func triggerCollabRepair(s *Server, c *gin.Context) {
	var opRequest CollaborativeRepairOperationRequest
	if err := c.ShouldBindJSON(&opRequest); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	newOp := &CollaborativeRepairOperation{
		FileCID:  opRequest.FileCID,
		MetaCID:  opRequest.MetaCID,
		Depth:    opRequest.Depth,
		Origin:   opRequest.Origin,
		NumPeers: opRequest.NumPeers,
	}

	s.collabOps <- newOp

	c.JSON(200, gin.H{"message": "Collab repair triggered"})
}

func triggerUnitRepair(s *Server, c *gin.Context) {
	var opRequest UnitRepairOperationRequest
	if err := c.ShouldBindJSON(&opRequest); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	newOp := &UnitRepairOperation{
		FileCID:       opRequest.FileCID,
		MetaCID:       opRequest.MetaCID,
		FailedIndices: opRequest.FailedIndices,
		Depth:         opRequest.Depth,
		Origin:        opRequest.Origin,
	}

	s.unitOps <- newOp

	c.JSON(200, gin.H{"message": "Unit repair triggered"})

}

func triggerStrandRepair(s *Server, c *gin.Context) {
	var opRequest StrandRepairOperationRequest
	if err := c.ShouldBindJSON(&opRequest); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	newOp := &StrandRepairOperation{
		FileCID: opRequest.FileCID,
		MetaCID: opRequest.MetaCID,
		Strand:  opRequest.Strand,
		Depth:   opRequest.Depth,
	}

	s.strandOps <- newOp

	c.JSON(200, gin.H{"message": "Strand repair triggered"})
}

func reportUnitRepair(s *Server, c *gin.Context) {
	var opResponse UnitRepairOperationResponse
	if err := c.ShouldBindJSON(&opResponse); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	newOp := &UnitRepairDone{
		FileCID:      opResponse.FileCID,
		MetaCID:      opResponse.MetaCID,
		Origin:       opResponse.Origin,
		RepairStatus: opResponse.RepairStatus,
	}

	s.unitDone <- newOp
}

func reportCollabRepair(s *Server, c *gin.Context) {
	var opResponse CollaborativeRepairOperationResponse
	if err := c.ShouldBindJSON(&opResponse); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	newOp := &CollaborativeRepairDone{
		FileCID:      opResponse.FileCID,
		MetaCID:      opResponse.MetaCID,
		Origin:       opResponse.Origin,
		RepairStatus: opResponse.RepairStatus,
	}

	s.collabDone <- newOp
}
