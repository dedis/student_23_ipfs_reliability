package Server

import "math/rand"

// ComputeHealth
// @Description: Computes the estimated health of the file, corresponding to repairability of the file.
// This value is based on the missing blocks and the current state of the cluster.
func (fs *FileStats) ComputeHealth() float32 {
	// TODO: Update (repairability based on lattice with missing blocks and estimated block probability)
	// sample three diff lattices with the missing blocks and the estimated block probability
	// then compute the average repairability of the file (using the max repair depth of each)
	return fs.EstimatedBlockProb
}

// selectBlockHeuristic
// @Description: Selects a block to inspect based on the current view of the file.
func (s *Server) selectBlockHeuristic(fs *FileStats) (uint, bool) {
	// TODO: Make usage of missing blocks and cluster state (region, etc.)

	// 1/2 chance to select a data block
	// 		1/4 chance to retry a missing block
	//      2/4 chance to try using tag:region to find new missing blocks
	//      1/4 chance to select a block at random
	// 1/2 chance to select a parity block
	//      same...

	return uint(rand.Intn(fs.NumDataBlocks)), true
}

// InspectFile
// @Description: Inspect a block of the file (parity or data) and update the stats
func (s *Server) InspectFile(fileCID string, fs *FileStats) {

	// select block heuristically
	blockNumber, isData := s.selectBlockHeuristic(fs)

	// TODO get CID from Block number
	var blockCID string
	if isData {

	} else {

	}

	// check block
	_, err := s.client.GetRawBlock(blockCID)
	// TODO: timeout needed?

	if err == nil {
		fs.updateBlockProb(1.0)
		delete(fs.DataBlocksMissing, blockNumber)
	} else {
		watchedBlock, in := fs.DataBlocksMissing[blockNumber]
		if in {
			watchedBlock.Probability /= 3
		} else {
			watchedBlock.CID = blockCID
			watchedBlock.Probability = 0.33

			allocations, err := s.client.IPFSClusterConnector.GetPinAllocations(blockCID)
			if err != nil {
				return
			}
			watchedBlock.Peer.Name = allocations[0]
			// TODO: fetch region of this peer if possible (from metrics... maybe impl function for this)
			// watchedBlock.Peer.Region = ...
			s.state.potentialFailedRegions[watchedBlock.Peer.Region] = append(s.state.potentialFailedRegions[watchedBlock.Peer.Region], watchedBlock.Peer.Name)

			fs.DataBlocksMissing[blockNumber] = watchedBlock
		}

		// FIXME: Too harsh?
		fs.updateBlockProb(watchedBlock.Probability)

		fs.Health = fs.ComputeHealth()
		if fs.Health < s.repairThreshold {
			// TODO trigger repair
		}
	}
}

func (fs *FileStats) updateBlockProb(testedBlockProb float32) {
	fs.EstimatedBlockProb = (fs.EstimatedBlockProb*float32(fs.NumDataBlocks+fs.NumParityBlocks-1) + testedBlockProb) / float32(fs.NumDataBlocks+fs.NumParityBlocks)

}
