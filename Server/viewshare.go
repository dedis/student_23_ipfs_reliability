package Server

func (s *Server) ShareView(fileCID string, fs *FileStats) {
	// TODO: impl
	// Check allocation list for fs.strandRootCID
	//   send a view of the stats to each CommunityNode corresponding to a peer in the allocation list
}

func (s *Server) UpdateView(fileCID string, fs *FileStats) {
	// TODO: impl
	// Update stats based on the view received from another CommunityNodes
	// 	include missing blocks to stats
	//	update estimatedBlockProb (by taking the average)
	//  recompute health
}
