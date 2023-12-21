package Server

import (
	"encoding/json"
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

				var request StartMonitoringRequest
				// parse args
				if err := json.Unmarshal(op.parameter, &request); err != nil {
					println("Error parsing StartMonitoringRequest: ", err.Error())
					continue
				}

				s.stateMux.Lock()
				_, in := s.state.files[request.FileCID]
				if !in {
					// If not already monitoring this file

					// get lattice and find strand number
					_, _, lattice, _, _, err := s.client.PrepareRepair(request.FileCID, request.MetadataCID, 2)
					// TODO is 5th return value required? (see after testing) = indexNodeMap (replaced it with Getter.GetData/ParityCID)

					if err != nil {
						println("Error in PrepareRepair: ", err.Error())
						continue
					}

					metaData, err := s.client.GetMetaData(request.MetadataCID)

					strandNumber := 0
					for i, root := range metaData.TreeCIDs {
						if root == request.StrandRootCID {
							strandNumber = i
							break
						}
					}

					s.state.files[request.FileCID] = &FileStats{request.FileCID, request.MetadataCID, request.StrandRootCID,
						strandNumber, lattice, make(map[uint]WatchedBlock),
						make(map[uint]WatchedBlock), make(map[uint]WatchedBlock),
						make(map[uint]WatchedBlock), 1.0, 1.0}
				}
				s.stateMux.Unlock()

			case op.operationType == STOP_MONITOR_FILE:

				var request StopMonitoringRequest
				// parse args
				if err := json.Unmarshal(op.parameter, &request); err != nil {
					println("Error parsing StartMonitoringRequest: ", err.Error())
					continue
				}
				s.stateMux.Lock()
				delete(s.state.files, request.FileCID)
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
				s.InspectFile(stats)
			}
			s.stateMux.Unlock()

			timerFiles.Reset(30 * time.Second)

		case <-timerShareView.C:
			s.stateMux.Lock()

			// share view for each file
			for file, stats := range s.state.files {
				println("Sharing view for file: ", file, "with strandRoot: ", stats.StrandRootCID, "\n")

				// Check allocation list for fs.strandRootCID
				//   send a view of the stats to each CommunityNode corresponding to a peer in the allocation list
				s.ShareView(file, stats)
			}
			s.stateMux.Unlock()

			timerShareView.Reset(3 * time.Minute)
		}
	}
}
