package Server

import (
	"github.com/gin-gonic/gin"
)

type State struct {
	// TODO: Add state
}

type Operation struct {
	// TODO: Define
}

type Server struct {
	ginEngine  *gin.Engine
	state      State
	operations chan Operation
	ctx        chan struct{}
}

func (s *Server) setUpServer() {
	s.ginEngine = gin.Default()
	// TODO: Config port and other network settings

	s.ginEngine.POST("/startMonitor", func(c *gin.Context) { startMonitor(s, c) })
	s.ginEngine.POST("/stopMonitor", func(c *gin.Context) { stopMonitor(s, c) })
	s.ginEngine.GET("/listMonitor", func(c *gin.Context) { listMonitor(s, c) })
	s.ginEngine.GET("/checkFileStatus", func(c *gin.Context) { checkFileStatus(s, c) })
	s.ginEngine.GET("/checkClusterStatus", func(c *gin.Context) { checkClusterStatus(s, c) })
	s.ginEngine.POST("/notifyFailure", func(c *gin.Context) { notifyFailure(s, c) })

	// TODO: Init State
}

// RunServer
// * @Description: Run the server (blocking)
// * @param port: The port to listen on
func (s *Server) RunServer(port string) int {
	s.setUpServer()

	// Starting daemon
	go Daemon(s)
	defer close(s.ctx)

	err := s.ginEngine.Run(port) // blocking
	// listen and serve on 0.0.0.0 + port
	if err != nil {
		return 1
	}

	return 0
}

func startMonitor(s *Server, c *gin.Context) {
	// TODO implement
}

func stopMonitor(s *Server, c *gin.Context) {
	// TODO implement
}

func listMonitor(s *Server, c *gin.Context) {
	// TODO implement
}

func checkFileStatus(s *Server, c *gin.Context) {
	// TODO implement
}

func checkClusterStatus(s *Server, c *gin.Context) {
	// TODO implement
}

func notifyFailure(s *Server, c *gin.Context) {
	// TODO implement
}
