package Server

import (
	"encoding/json"
	"time"
)

const InspectionInterval = 30 * time.Second
const ViewSharingInterval = 4 * time.Minute

func Daemon(s *Server) {
	timerFiles := time.NewTimer(InspectionInterval)
	timerShareView := time.NewTimer(ViewSharingInterval)

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

					s.RefreshClient()
					s.client.SetTimeout(5 * time.Second)

					metaData, err := s.client.GetMetaData(request.MetadataCID)
					if err != nil {
						println("Could not fetch the metadata: ", err.Error())
						continue
					}

					strandNumber := 0
					for i, root := range metaData.TreeCIDs {
						if root == request.StrandRootCID {
							strandNumber = i
							break
						}
					}

					s.state.files[request.FileCID] = &FileStats{request.FileCID, request.MetadataCID, request.StrandRootCID,
						strandNumber, make(map[uint]*WatchedBlock), make(map[uint]*WatchedBlock),
						make(map[uint]*WatchedBlock), 1.0, 1.0}
				}
				s.stateMux.Unlock()

			case op.operationType == STOP_MONITOR_FILE:

				var request StopMonitoringRequest
				// parse args
				if err := json.Unmarshal(op.parameter, &request); err != nil {
					println("Error parsing StopMonitoringRequest: ", err.Error())
					continue
				}
				s.stateMux.Lock()
				delete(s.state.files, request.FileCID)
				s.stateMux.Unlock()

			case op.operationType == RESET_MONITOR_FILE:

				var request ResetMonitoringRequest
				// parse args
				if err := json.Unmarshal(op.parameter, &request); err != nil {
					println("Error parsing StopMonitoringRequest: ", err.Error())
					continue
				}

				s.stateMux.Lock()

				parityBlocksMissing := make(map[uint]*WatchedBlock)
				validParityBlocksHistory := make(map[uint]*WatchedBlock)
				dataBlocksMissing := make(map[uint]*WatchedBlock)

				blocProb := s.state.files[request.FileCID].EstimatedBlockProb
				health := s.state.files[request.FileCID].Health

				// Keep old values for parity blocks if only data blocks were repaired
				if request.IsData {
					parityBlocksMissing = s.state.files[request.FileCID].ParityBlocksMissing
					validParityBlocksHistory = s.state.files[request.FileCID].validParityBlocksHistory
				} else {
					dataBlocksMissing = s.state.files[request.FileCID].DataBlocksMissing
				}

				s.state.files[request.FileCID] = &FileStats{
					fileCID:                  request.FileCID,
					MetadataCID:              s.state.files[request.FileCID].MetadataCID,
					StrandRootCID:            s.state.files[request.FileCID].StrandRootCID,
					strandNumber:             s.state.files[request.FileCID].strandNumber,
					DataBlocksMissing:        dataBlocksMissing,
					ParityBlocksMissing:      parityBlocksMissing,
					validParityBlocksHistory: validParityBlocksHistory,
					EstimatedBlockProb:       (blocProb + 1) / 2,
					Health:                   (health + 1) / 2,
				}

				s.stateMux.Unlock()

			default:
				println("Unknown operation type (", op.operationType, "), please fix...")
			}
			break

		case <-timerFiles.C:
			s.stateMux.Lock()
			s.RefreshClient()
			s.client.SetTimeout(5 * time.Second)

			// check a block for each file
			for file, stats := range s.state.files {
				println("Checking file: ", file, "with strandRoot: ", stats.StrandRootCID, "\n")
				s.InspectFile(stats)
			}
			s.stateMux.Unlock()

			timerFiles.Reset(InspectionInterval)
			s.client.SetTimeout(0)

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

			timerShareView.Reset(ViewSharingInterval)
		}
	}
}
