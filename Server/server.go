package Server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
	"strconv"
	"strings"
)

func (s *Server) setUpServer() {
	s.ginEngine = gin.Default()
	// TODO: Config port and other network settings

	s.ginEngine.POST("/startMonitorFile", func(c *gin.Context) { startMonitorFile(s, c) })
	s.ginEngine.POST("/stopMonitorFile", func(c *gin.Context) { stopMonitorFile(s, c) })
	s.ginEngine.GET("/listMonitor", func(c *gin.Context) { listMonitor(s, c) })
	s.ginEngine.GET("/checkFileStatus", func(c *gin.Context) { checkFileStatus(s, c) })
	s.ginEngine.GET("/checkClusterStatus", func(c *gin.Context) { checkClusterStatus(s, c) })
	s.ginEngine.POST("/notifyFailure", func(c *gin.Context) { notifyFailure(s, c) })

	// TODO: Init State

	s.ctx = make(chan struct{})
	s.operations = make(chan Operation)
	s.state = State{files: make(map[string]FileStats)}
	s.sh = shell.NewLocalShell() // FIXME need to call NewShell?
}

// RunServer
// @Description: Run the server (blocking)
// @param port: The port to listen on
func (s *Server) RunServer(port int) int {
	s.setUpServer()

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

	s.operations <- Operation{START_MONITOR_FILE, dataRoot}

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

func notifyFailure(s *Server, c *gin.Context) {
	// TODO implement
	c.JSON(503, gin.H{"message": "Not Yet implemented"})
}
