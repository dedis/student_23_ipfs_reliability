package Server

import (
	"strconv"
	"strings"
	"time"
)

func Daemon(s *Server) {
	timerFiles := time.NewTimer(30 * time.Second)
	timerShareView := time.NewTimer(3 * time.Minute)

	for {
		select {
		case <-s.ctx:
			return
		case op := <-s.collabOps:
			s.StartCollabRepair(op)
		case op := <-s.unitOps:
			s.StartUnitRepair(op)
		case op := <-s.collabDone:
			s.ContinueStrandRepair(op)
		case op := <-s.unitDone:
			s.ReportUnitRepair(op)
		case op := <-s.strandOps:
			s.StartStrandRepair(op)

		case op := <-s.operations:
			switch {
			case op.operationType == START_MONITOR_FILE:
				res := strings.Split(op.parameter, ",")
				if len(res) != 3 {
					println("Incorrect number of parameters for START_MONITOR_FILE")
					break
				}
				dataCID := res[0]
				strandCID := res[1]
				numDataBlocks, err := strconv.Atoi(res[2])
				if err != nil {
					println("Number of blocks not an int in START_MONITOR_FILE")
					break
				}
				numParityBlocks, err := strconv.Atoi(res[3])
				if err != nil {
					println("Number of blocks not an int in START_MONITOR_FILE")
					break
				}

				s.stateMux.Lock()
				s.state.files[dataCID] = FileStats{strandCID, numDataBlocks,
					make(map[uint]WatchedBlock), numParityBlocks,
					make(map[uint]WatchedBlock), 1.0, 1.0}
				s.stateMux.Unlock()

			case op.operationType == STOP_MONITOR_FILE:
				s.stateMux.Lock()
				delete(s.state.files, op.parameter)
				s.stateMux.Unlock()

			default:
				println("Unknown operation type (", op.operationType, "), please fix...")
			}
			break

		case <-timerFiles.C:
			s.stateMux.Lock()
			// check a block for each file
			for file, stats := range s.state.files {
				println("Checking file: ", file, "with strandRoot: ", stats.StrandRootCID, "\n")
				s.InspectFile(file, &stats)
			}
			s.stateMux.Unlock()

			timerFiles.Reset(30 * time.Second)

		case <-timerShareView.C:
			s.stateMux.Lock()

			// share view for each file
			for file, stats := range s.state.files {
				println("Sharing view for file: ", file, "with strandRoot: ", stats.StrandRootCID, "\n")
				s.ShareView(file, &stats)
			}
			s.stateMux.Unlock()

			timerShareView.Reset(3 * time.Minute)
		}
	}
}
