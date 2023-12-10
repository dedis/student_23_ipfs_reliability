package Server

import (
	"strings"
	"time"
)

func Daemon(s *Server) {
	timer := time.NewTicker(30 * time.Second) // FIXME goto timer if ops take some time to be processed

	for {
		select {
		case <-s.ctx:
			return

		case op := <-s.operations:
			switch {
			case op.operationType == START_MONITOR_FILE:
				res := strings.Split(op.parameter, ",")
				if len(res) != 2 {
					println("Incorrect number of parameters for START_MONITOR_FILE")
					break
				}
				dataCID := res[0]
				strandCID := res[1]

				s.stateMux.Lock()
				s.state.files[dataCID] = FileStats{strandCID}
				s.stateMux.Unlock()

			case op.operationType == STOP_MONITOR_FILE:
				s.stateMux.Lock()
				delete(s.state.files, op.parameter)
				s.stateMux.Unlock()

			case op.operationType == REPARE_FILE:
				// TODO collaborative repairs from here? or done in a go routine somewhere else?
				println("Collaborative repair not implemented yet")

			default:
				println("Unknown operation type, please fix...")
			}
			break

		case <-timer.C:
			s.stateMux.Lock()
			// TODO: Periodic operations
			println("Periodic operations... (TODO)\n")

			// TODO: Check a block for each file
			for file, stats := range s.state.files {
				println("Checking file", file, "with strand", stats.strandCID, "... (TODO)\n")
			}

			// TODO: Check Cluster health?
			s.stateMux.Unlock()
		}
	}
}
