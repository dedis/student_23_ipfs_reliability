package Server

import "github.com/gin-gonic/gin"

type State struct {
	// TODO: Add state
}

type Server struct {
	ginEngine *gin.Engine
	state     State
}

func (s *Server) setUpServer() {
	s.ginEngine = gin.Default()
	// TODO: Config port and other network settings

	s.ginEngine.POST("/startMonitor", startMonitor)
	s.ginEngine.POST("/stopMonitor", stopMonitor)
	s.ginEngine.GET("/listMonitor", listMonitor)
	s.ginEngine.GET("/checkFileStatus", checkFileStatus)
	s.ginEngine.GET("/checkClusterStatus", checkClusterStatus)
	s.ginEngine.POST("/notifyFailure", notifyFailure)

	// TODO: Init State
}

// RunServer
// * @Description: Run the server (blocking)
func (s *Server) RunServer() int {
	s.setUpServer()

	err := s.ginEngine.Run()
	// listen and serve on 0.0.0.0:8080
	if err != nil {
		return 1
	}

	return 0
}

func startMonitor(c *gin.Context) {
	// TODO implement
}

func stopMonitor(c *gin.Context) {
	// TODO implement
}

func listMonitor(c *gin.Context) {
	// TODO implement
}

func checkFileStatus(c *gin.Context) {
	// TODO implement
}

func checkClusterStatus(c *gin.Context) {
	// TODO implement
}

func notifyFailure(c *gin.Context) {
	// TODO implement
}
