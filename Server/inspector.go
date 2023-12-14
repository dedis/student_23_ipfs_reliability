package Server

import "math/rand"

// Health
// @Description: Computes the estimated health of the file, corresponding to repairability of the file.
// This value is based on the missing blocks and the current state of the cluster.
func (fs *FileStats) Health() float32 {
	// TODO: Update (repairability based on lattice with missing blocks and estimated block probability)
	return fs.EstimatedBlockProb
}

// selectBlockHeuristic
// @Description: Selects a block to inspect based on the current view of the file.
func (s *Server) selectBlockHeuristic(fs *FileStats) uint {
	// TODO: Make usage of missing blocks and cluster state (region, etc.)

	// 1/2 chance to select a data block
	// 		1/4 chance to retry a missing block
	//      2/4 chance to try using tag:region to find new missing blocks
	//      1/4 chance to select a block at random
	// 1/2 chance to select a parity block
	//      same...

	return uint(rand.Intn(fs.numDataBlocks))
}

// InspectFile
// @Description: Inspect a block of the file (parity or data) and update the stats
func (s *Server) InspectFile(fileCID string, fs *FileStats) {

	// select block heuristically
	blockNumber := s.selectBlockHeuristic(fs)

	// TODO get CID from Block number
	blockCID := fileCID // FIXME <-

	// check block
	//_, err := s.sh.BlockGet(blockCID)
	var err error
	err = nil
	// TODO: timeout needed?

	if err == nil {
		fs.EstimatedBlockProb = (fs.EstimatedBlockProb*float32(fs.numDataBlocks+fs.numParityBlocks-1) + 1) / float32(fs.numDataBlocks+fs.numParityBlocks)
		delete(fs.DataBlocksMissing, blockNumber)
	} else {
		watchedBlock, in := fs.DataBlocksMissing[blockNumber]
		if in {
			watchedBlock.Probability /= 3
		} else {
			watchedBlock.CID = blockCID
			watchedBlock.Probability = 0.33

			// TODO find Peer responsible for this block | check allocation for this CID
			fs.DataBlocksMissing[blockNumber] = watchedBlock
		}

		// FIXME: Too harsh?
		fs.EstimatedBlockProb = (fs.EstimatedBlockProb*float32(fs.numDataBlocks+fs.numParityBlocks-1) + watchedBlock.Probability) / float32(fs.numDataBlocks+fs.numParityBlocks)

		// TODO recompute health
	}

	// TODO update stats about Cluster?

}
