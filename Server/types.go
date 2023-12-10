package Server

import (
	"github.com/gin-gonic/gin"
	"sync"
)

type State struct {
	files map[string]FileStats
	// TODO: Add state
}

type FileStats struct {
	strandCID string
	// TODO: Add stats
}

type LatticeView struct {
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
	// TODO: Define
}

type Server struct {
	ginEngine  *gin.Engine
	state      State
	stateMux   sync.Mutex
	operations chan Operation
	ctx        chan struct{}
}
