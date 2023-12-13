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
				numBlocks, err := strconv.Atoi(res[2])
				if err != nil {
					println("Number of blocks not an int in START_MONITOR_FILE")
					break
				}

				s.stateMux.Lock()
				s.state.files[dataCID] = FileStats{strandCID, numBlocks,
					make(map[uint]WatchedBlock), make(map[uint]WatchedBlock),
					0.1, 1.0}
				s.stateMux.Unlock()

			case op.operationType == STOP_MONITOR_FILE:
				s.stateMux.Lock()
				delete(s.state.files, op.parameter)
				s.stateMux.Unlock()

			default:
				println("Unknown operation type, please fix...")
			}
			break

		case <-timerFiles.C:
			s.stateMux.Lock()
			// check a block for each file
			for file, stats := range s.state.files {
				println("Checking file: ", file, "with strandRoot: ", stats.strandRootCID, "\n")
				s.InspectFile(file, &stats)
			}
			s.stateMux.Unlock()

			timerFiles.Reset(30 * time.Second)

		case <-timerShareView.C:
			s.stateMux.Lock()

			// TODO: for each StrandRootCID that the node tracks, send view of stats to other trackers
			//  make others track if re-pin operation occurred
			for file, stats := range s.state.files {
				println("Sharing view for file: ", file, "with strandRoot: ", stats.strandRootCID, "\n")
				s.ShareView(file, &stats)
			}
			s.stateMux.Unlock()

			timerShareView.Reset(3 * time.Minute)
		}
	}
}
