package Server

import (
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
				if len(res) != 2 {
					println("Incorrect number of parameters for START_MONITOR_FILE")
					break
				}
				dataCID := res[0]
				strandCID := res[1]

				s.stateMux.Lock()
				s.state.files[dataCID] = FileStats{strandCID,
					make(map[uint]WatchedBlock),
					make(map[uint]WatchedBlock), make(map[uint]WatchedBlock),
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

				// TODO: impl
				// Check allocation list for fs.strandRootCID
				//   send a view of the stats to each CommunityNode corresponding to a peer in the allocation list
				s.ShareView(file, &stats)
				// If not in the allocation list, stop tracking the file

			}
			s.stateMux.Unlock()

			timerShareView.Reset(3 * time.Minute)
		}
	}
}
