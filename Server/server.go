package Server

import (
	"encoding/json"
	"fmt"
	"ipfs-alpha-entanglement-code/client"
	"ipfs-alpha-entanglement-code/util"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const HealthRepairThreshold = 0.6

// SetUpServer
// @Description: Initialize the Server struct and the REST endpoints
func (s *Server) setUpServer() {
	s.ginEngine = gin.Default()

	// monitoring endpoints
	s.ginEngine.POST("/forwardMonitoring", func(c *gin.Context) { forwardMonitoring(s, c) })
	s.ginEngine.POST("/startMonitorFile", func(c *gin.Context) { startMonitorFile(s, c) })
	s.ginEngine.POST("/stopMonitorFile", func(c *gin.Context) { stopMonitorFile(s, c) })
	s.ginEngine.POST("/resetMonitorFile", func(c *gin.Context) { resetMonitorFile(s, c) })
	s.ginEngine.GET("/listMonitor", func(c *gin.Context) { listMonitor(s, c) })
	s.ginEngine.GET("/checkFileStatus", func(c *gin.Context) { checkFileStatus(s, c) })
	s.ginEngine.GET("/checkClusterStatus", func(c *gin.Context) { checkClusterStatus(s, c) })
	s.ginEngine.POST("/updateView", func(c *gin.Context) { prepareUpdateView(s, c) })
	s.ginEngine.GET("/recomputeHealth", func(c *gin.Context) { recomputeHealth(s, c) })

	// repair endpoints
	s.ginEngine.GET("/downloadFile", func(c *gin.Context) { downloadFile(s, c) })
	s.ginEngine.POST("/triggerCollabRepair", func(c *gin.Context) { triggerCollabRepair(s, c) })
	s.ginEngine.POST("/triggerUnitRepair", func(c *gin.Context) { triggerUnitRepair(s, c) })
	s.ginEngine.POST("/triggerStrandRepair", func(c *gin.Context) { triggerStrandRepair(s, c) })
	s.ginEngine.POST("/reportUnitRepair", func(c *gin.Context) { reportUnitRepair(s, c) })
	s.ginEngine.POST("/reportCollabRepair", func(c *gin.Context) { reportCollabRepair(s, c) })

	s.ginEngine.GET("/health-check", func(c *gin.Context) { c.Status(200) })

	// init state
	s.ctx = make(chan struct{})
	s.operations = make(chan Operation)
	s.state = State{files: make(map[string]*FileStats),
		potentialFailedRegions:      make(map[string][]string),
		unavailableBlocksTimestamps: make([]int64, 0)}
	s.state.unavailableBlocksTimestamps = append(s.state.unavailableBlocksTimestamps, time.Now().UnixNano())
	serverClient, err := client.NewClient(s.clusterIP, s.clusterPort, s.ipfsIP, s.ipfsPort)
	if err != nil {
		log.Println("Error creating Server client: ", err)
	}
	s.client = serverClient
	s.repairThreshold = HealthRepairThreshold
	// s.ipConverter = &docker.DockerClusterToCommunityConverter{}
	s.collabOps = make(chan *CollaborativeRepairOperation)
	s.collabDone = make(chan *CollaborativeRepairDone)
	s.unitOps = make(chan *UnitRepairOperation)
	s.unitDone = make(chan *UnitRepairDone)
	s.strandOps = make(chan *StrandRepairOperation)
	s.collabData = make(map[string]*CollaborativeRepairData)
	s.strandData = make(map[string]*StrandRepairData)
}

func (s *Server) AnnounceSelf() error {
	// announce self to discovery server
	// send an http post request to discovery server
	/*
			takes the following as parameters:

			communityIP
		    clusterIP
		    clusterPort
		    ipfsIP
		    ipfsPort

	*/

	baseURL := fmt.Sprintf("http://%s/announce", s.discoveryAddress)

	body := &CommunityNodeAnnouncement{
		CommunityIP: s.address,
		ClusterIP:   s.clusterIP,
		ClusterPort: s.clusterPort,
		IpfsIP:      s.ipfsIP,
		IpfsPort:    s.ipfsPort,
	}

	jsonResponse, err := json.Marshal(body)
	if err != nil {
		util.LogPrintf("Error in marshalling body for community node announcement %s - %s", s.address, err)
	}

	// send the response back to the origin
	status, err := PostJSON(baseURL, jsonResponse)

	if err != nil {
		return fmt.Errorf("error announcing oneself: %v", err)
	}

	if status != 200 {
		return fmt.Errorf("error announcing oneself, status not 200 but: %d", status)
	}

	return nil
}

// RunServer
// @Description: Run the server (blocking)
// @param port: The port to listen on
func (s *Server) RunServer(port int, communityIP string, clusterIP string, clusterPort int, IpfsIP string, IpfsPort int, discovery string) int {
	s.clusterIP = clusterIP
	s.clusterPort = clusterPort
	s.ipfsIP = IpfsIP
	s.ipfsPort = IpfsPort
	s.discoveryAddress = discovery

	s.setUpServer()

	s.address = fmt.Sprintf("%s:%d", communityIP, port)
	util.LogPrintf("Server listening on %s", s.address)

	// announce self to discovery server
	err := s.AnnounceSelf()
	if err != nil {
		util.LogPrintf("Error announcing self: %v", err)
		return 1
	}

	// Starting daemon
	go Daemon(s)
	defer close(s.ctx)

	err = s.ginEngine.Run(fmt.Sprintf(":%d", port)) // blocking
	// listen and serve on 0.0.0.0 + port
	if err != nil {
		return 1
	}

	return 0
}

func forwardMonitoring(s *Server, c *gin.Context) {
	var monitoringRequest ForwardMonitoringRequest
	// parse args
	if err := c.ShouldBindJSON(&monitoringRequest); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	monitoringRequest = ForwardMonitoringRequest{
		FileCID:        monitoringRequest.FileCID,
		MetadataCID:    monitoringRequest.MetadataCID,
		StrandRootCIDs: monitoringRequest.StrandRootCIDs,
	}

	s.RefreshClient()
	s.client.SetTimeout(2 * time.Second)
	defer s.client.SetTimeout(0)

	time.Sleep(5 * time.Second) // give time to allocation to succeed

	// for strandRoot in strandCIDs: -> peers = c.IPFSClusterConnector.GetPinAllocations(strandRoot)
	for _, strandRoot := range monitoringRequest.StrandRootCIDs {
		peers, err := s.client.IPFSClusterConnector.GetPinAllocations(strandRoot)

		if err != nil {
			log.Printf("Couldn't start tracking for root CID: %s\n", strandRoot)
			continue
		}
		// for peer in peers: -> send peer's Community Node [startTracking FileCID - strandRoot]
		if len(peers) == 0 {
			log.Println("No peer found to be storing this strandRootCID: ", strandRoot)
			continue
		}

		request := StartMonitoringRequest{
			FileCID:       monitoringRequest.FileCID,
			MetadataCID:   monitoringRequest.MetadataCID,
			StrandRootCID: strandRoot,
		}

		body, err := json.Marshal(request)

		if err != nil {
			log.Println("Error marshalling request: ", err)
			continue
		}

		for _, peer := range peers {
			if peer == "" {
				log.Printf("Skiping peer with empty name *")
				continue
			}

			communityPeerAddress, err := s.getCommunityAddress(peer)
			if communityPeerAddress == "" {
				log.Printf("Could not find community address for peer: %s\n", peer)
				continue
			}

			if err != nil {
				log.Printf("Skiping peer: %s for file: %s [err=%v]\n", peer, monitoringRequest.FileCID, err)
				continue
			}

			status, err := PostJSON("http://"+communityPeerAddress+fmt.Sprintf("/startMonitorFile"), body)
			if err != nil {
				log.Println("Status: ", status, "Error: ", err)
			} else {
				log.Printf("Forwarded monitoring request for file: %s to peer: %s\n", monitoringRequest.FileCID, peer)
			}
		}
	}
}

// startMonitorFile
// Query parameters in context: dataRoot (CID-string), strandParityRoot (CID-string), numBlocks (int)
func startMonitorFile(s *Server, c *gin.Context) {
	var request StartMonitoringRequest
	// parse args
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	param, err := json.Marshal(request)
	if err != nil {
		c.JSON(400, gin.H{"message": "Malformated parameters"})
		return
	}
	s.operations <- Operation{START_MONITOR_FILE, param}

	c.JSON(200, gin.H{"message": "Start op."})
}

// stopMonitorFile
// Query parameters in context: dataRoot (CID-string)
func stopMonitorFile(s *Server, c *gin.Context) {
	var request StopMonitoringRequest
	// parse args
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	param, err := json.Marshal(request)
	if err != nil {
		c.JSON(400, gin.H{"message": "Malformated parameters"})
		return
	}

	s.operations <- Operation{STOP_MONITOR_FILE, param}

	c.JSON(200, gin.H{"message": "Stop op."})
}

func resetMonitorFile(s *Server, c *gin.Context) {
	var request ResetMonitoringRequest
	// parse args
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	param, err := json.Marshal(request)
	if err != nil {
		c.JSON(400, gin.H{"message": "Malformated parameters"})
		return
	}

	s.operations <- Operation{RESET_MONITOR_FILE, param}

	c.JSON(200, gin.H{"message": "Reset op."})
}

// resetMonitorFile
// reset stats for file (fileCID) after repair
func (s *Server) resetMonitorFile(fileCID string, isData bool) {
	request := ResetMonitoringRequest{
		FileCID: fileCID,
		IsData:  isData,
	}

	param, err := json.Marshal(request)
	if err != nil {
		log.Println("Error marshalling request: ", err)
		return
	}

	// send reset op. to all monitor nodes for this file
	metaData, err := s.client.GetMetaData(s.state.files[fileCID].MetadataCID)

	if err != nil {
		println("Could not fetch the metadata: ", err.Error())
		return
	}

	for _, root := range metaData.TreeCIDs {
		peers, err := s.client.IPFSClusterConnector.GetPinAllocations(root)

		if err != nil {
			log.Printf("Couldn't start tracking for root CID: %s\n", root)
			continue
		}

		for _, peer := range peers {
			if peer == "" {
				log.Printf("Skiping peer with empty name")
				continue
			}

			communityPeerAddress, err := s.getCommunityAddress(peer)
			if err != nil {
				log.Printf("Skiping peer: %s for file: %s [err=%v]\n", peer, fileCID, err)
				continue
			}

			status, err := PostJSON("http://"+communityPeerAddress+fmt.Sprintf("/resetMonitorFile"), param)
			if err != nil {
				log.Println("Status: ", status, "Error: ", err)
			}
		}
	}

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
		cids += " - CID=" + file + "-[BlockProb = " +
			strconv.FormatFloat(float64(stats.EstimatedBlockProb*100), 'f', -1, 64) + "%]\n"
	}

	c.JSON(200, gin.H{"Result": "Listing monitored CIDs: " + cids[:len(cids)-1]})
}

// checkFileStatus
// Query parameters in context: fileCID (CID-string)
func checkFileStatus(s *Server, c *gin.Context) {
	fileCID := c.Query("fileCID")
	if fileCID == "" {
		c.JSON(400, gin.H{"message": "Invalid CID parameter"})
		return
	}

	stats, in := s.state.files[fileCID]

	if !in {
		c.JSON(400, gin.H{"message": "File not monitored or invalid CID"})
		return
	}

	// Pack stats in string
	ret := "=====================================\n"
	ret += " * FileCID: " + stats.fileCID + "\n"
	ret += " * MetadataCID: " + stats.MetadataCID + "\n"
	ret += " * StrandRootCID: " + stats.StrandRootCID + "\n"
	ret += " * StrandNumber: " + strconv.Itoa(stats.strandNumber) + "\n"
	ret += " * Nb of data blocks missing: " + fmt.Sprint(len(stats.DataBlocksMissing)) + "\n"
	ret += " * Nb of parity blocks missing: " + fmt.Sprint(len(stats.ParityBlocksMissing)) + "\n"
	ret += " * EstimatedBlockProb: " + strconv.FormatFloat(float64(stats.EstimatedBlockProb*100), 'f', -1, 64) + "%\n"
	ret += " * Health: " + strconv.FormatFloat(float64(stats.Health*100), 'f', -1, 64) + "%\n"
	ret += "=====================================\n"

	c.JSON(200, gin.H{"Result": ret})
}

// checkClusterStatus
func checkClusterStatus(s *Server, c *gin.Context) {
	// Pack state in string
	ret := "=====================================\n"
	ret += " * ClusterIP: " + s.clusterIP + "\n"
	ret += " * ClusterPort: " + strconv.Itoa(s.clusterPort) + "\n"
	ret += " * PotentialFailedRegions: " + fmt.Sprint(s.state.potentialFailedRegions) + "\n"

	cntMissing := len(s.state.unavailableBlocksTimestamps) - 1

	ret += " * Number detected missing blocks: " + fmt.Sprint(cntMissing) + "\n"

	if cntMissing > 0 {
		sumIntervals := 0
		for i := 0; i < cntMissing; i++ {
			sumIntervals += int(s.state.unavailableBlocksTimestamps[i+1] - s.state.unavailableBlocksTimestamps[i])
		}
		avgIntervalsNano := float64(sumIntervals) / float64(cntMissing)
		avgIntervals := time.Duration(avgIntervalsNano)

		ret += " * Average time between detected missing blocks: " + fmt.Sprint(avgIntervals) + " \n"
	}
	ret += "=====================================\n"
	c.JSON(200, gin.H{"Result": ret})
}

// Query parameters in context: fileCID (CID-string)
func prepareUpdateView(s *Server, c *gin.Context) {
	fileCID := c.Query("fileCID")

	var updateViewArgs FileStats
	// parse args
	if err := c.ShouldBindJSON(&updateViewArgs); err != nil {
		c.JSON(400, gin.H{"message": "Missing parameters"})
		return
	}

	updateViewArgs = FileStats{
		StrandRootCID:       updateViewArgs.StrandRootCID,
		DataBlocksMissing:   updateViewArgs.DataBlocksMissing,
		ParityBlocksMissing: updateViewArgs.ParityBlocksMissing,
		EstimatedBlockProb:  updateViewArgs.EstimatedBlockProb,
		Health:              updateViewArgs.Health,
	}

	s.UpdateView(fileCID, &updateViewArgs)
	c.JSON(503, gin.H{"message": "file view updated"})
}

// Query parameters in context: rootFileCID (CID-string), metadataCID (CID-string), path (string), uploadRecoverData (bool)
func downloadFile(s *Server, c *gin.Context) {
	startTime := time.Now()
	rootFileCID := c.Query("rootFileCID")
	metadataCID := c.Query("metadataCID")
	path := c.Query("path")
	uploadRecoverData := c.Query("uploadRecoverData")
	depth, err := strconv.Atoi(c.Query("depth"))
	if err != nil {
		depth = 1
	}

	s.RefreshClient()

	options := client.DownloadOption{
		UploadRecoverData: uploadRecoverData == "true",
		MetaCID:           metadataCID,
	}

	s.client.SetTimeout(100 * time.Millisecond)
	defer s.client.SetTimeout(0)
	status := PENDING
	data, getter, err := s.client.Download(rootFileCID, path, options, uint(depth))
	endTime := time.Now()

	if err != nil {
		c.Header("Content-Disposition", "attachment; filename="+path)
		c.Data(400, "application/octet-stream", []byte(err.Error()))
		status = FAILURE
	} else {
		c.Header("Content-Disposition", "attachment; filename="+path)
		c.Data(200, "application/octet-stream", data)
		status = SUCCESS
	}

	// Only report metrics if depth > 1 (actually doing some kind of repair)
	if depth > 1 {
		s.ReportDownloadMetrics(getter, &startTime, &endTime, status)
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
		FileCID:                 opResponse.FileCID,
		MetaCID:                 opResponse.MetaCID,
		Origin:                  opResponse.Origin,
		RepairStatus:            opResponse.RepairStatus,
		ParityAvailable:         opResponse.ParityAvailable,
		DataBlocksFetched:       opResponse.DataBlocksFetched,
		DataBlocksCached:        opResponse.DataBlocksCached,
		DataBlocksUnavailable:   opResponse.DataBlocksUnavailable,
		DataBlocksError:         opResponse.DataBlocksError,
		ParityBlocksFetched:     opResponse.ParityBlocksFetched,
		ParityBlocksCached:      opResponse.ParityBlocksCached,
		ParityBlocksUnavailable: opResponse.ParityBlocksUnavailable,
		ParityBlocksError:       opResponse.ParityBlocksError,
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

// recomputeHealth
// Query parameters in context: fileCID (CID-string)
func recomputeHealth(s *Server, c *gin.Context) {
	s.stateMux.Lock()
	defer s.stateMux.Unlock()

	fileCID := c.Query("fileCID")
	if fileCID == "" {
		c.JSON(400, gin.H{"message": "Invalid CID parameter"})
		return
	}
	stats, in := s.state.files[fileCID]
	if !in {
		c.JSON(400, gin.H{"message": "File not monitored or invalid CID"})
		return
	}

	s.RefreshClient()
	s.client.SetTimeout(1 * time.Second)

	_, _, lattice, _, _, err := s.client.PrepareRepair(fileCID, stats.MetadataCID, 2)

	if err != nil {
		println("Could not generate lattice: ", err.Error())
		return
	}

	health := s.ComputeHealth(stats, lattice)

	// Pack stats in string
	ret := "Health=" + strconv.FormatFloat(float64(health), 'f', -1, 64) + "\n"
	c.JSON(200, gin.H{"Result": ret})
}
