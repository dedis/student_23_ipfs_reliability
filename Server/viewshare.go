package Server

import (
	"encoding/json"
	"fmt"
	"log"
)

// ShareView
// @Description: broadcast view (stats) for a file to other monitors
func (s *Server) ShareView(fileCID string, fs *FileStats) {
	// Check allocation list for fs.strandRootCID
	//   send a view of the stats to each CommunityNode corresponding to a peer in the allocation list

	peers, err := s.client.IPFSClusterConnector.GetPinAllocations(fs.StrandRootCID)
	if err != nil {
		log.Println("Failed to share view for file: ", fileCID)
		return
	}
	// for peer in peers: -> send peer's Community Node [startTracking FileCID - strandRoot]
	log.Println("Test: len allocation peers = ", len(peers))

	body, err := json.Marshal(fs)
	if err != nil {
		log.Println("Failed to marshal view for file: ", fileCID)
		return
	}

	stillInPeers := false
	ownClusterName, _ := s.client.IPFSClusterConnector.PeerInfo()

	for _, peer := range peers {
		communityPeerAddress, err := s.getCommunityAddress(peer)
		if err != nil {
			log.Printf("Skiping peer: %s for file: %s\n", peer, fileCID)
			continue
		}

		status, err := PostJSON(communityPeerAddress+fmt.Sprintf("/updateView"), body)
		if err != nil {
			log.Println("Status: ", status)
		}
		if peer == ownClusterName { // TODO check if need to compare with s.clusterIP
			stillInPeers = true
		}
	}

	if !stillInPeers {
		// stop monitoring file
		request := StopMonitoringRequest{fileCID}
		body, err := json.Marshal(request)
		if err != nil {
			log.Println("Failed to marshal stop monitoring request for file: ", fileCID)
			return
		}

		status, err := PostJSON(s.address+fmt.Sprintf("/stopMonitorFile"), body)
		if err != nil {
			log.Println("Status: ", status)
		}
	}

}

// UpdateView
// @Description: Incorporates the view of another peer to own view for a file
func (s *Server) UpdateView(fileCID string, fs *FileStats) {
	s.stateMux.Lock()
	defer s.stateMux.Unlock()

	_, in := s.state.files[fileCID]

	if in {
		for i, dbm := range fs.DataBlocksMissing {
			s.state.files[fileCID].DataBlocksMissing[i] = dbm
		}
		for i, pbm := range fs.ParityBlocksMissing {
			s.state.files[fileCID].ParityBlocksMissing[i] = pbm
		}

		s.state.files[fileCID].EstimatedBlockProb = s.state.files[fileCID].EstimatedBlockProb + fs.EstimatedBlockProb/2

		s.state.files[fileCID].Health = s.ComputeHealth(s.state.files[fileCID])
		if fs.Health < s.repairThreshold {
			// TODO trigger repair (params?)
			s.repairFile(s.state.files[fileCID], 4, 2)
		}

	} else {
		// Start with these values
		request := StartMonitoringRequest{
			FileCID:       fileCID,
			MetadataCID:   fs.MetadataCID,
			StrandRootCID: fs.StrandRootCID,
		}
		body, err := json.Marshal(request)
		if err != nil {
			log.Println("Failed to marshal start monitoring request for file: ", fileCID)
			return
		}

		s.operations <- Operation{START_MONITOR_FILE, body}
	}
}
