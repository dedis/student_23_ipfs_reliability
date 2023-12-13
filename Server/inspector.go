package Server

import "math/rand"

// Health
// @Description: Computes the estimated health of the file, corresponding to repairability of the file.
// This value is based on the missing blocks and the current state of the cluster.
func (fs *FileStats) Health() float32 {
	// TODO: Update (repairability based on lattice with missing blocks and estimated block probability)
	return fs.estimatedBlockProb
}

// selectBlockHeuristic
// @Description: Selects a block to inspect based on the current view of the file.
func (s *Server) selectBlockHeuristic(fs *FileStats) uint {
	// TODO: Make usage of missing blocks and cluster state (region, etc.)
	return uint(rand.Intn(fs.numBlocks))
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
	// FIXME check if timeout tell: block was not there
	//  can store time (based on previous fetches) to download and set a timeout appropriately...

	if err == nil {
		fs.estimatedBlockProb = (fs.estimatedBlockProb*float32(fs.numBlocks-1) + 1) / float32(fs.numBlocks)
		delete(fs.dataBlocksMissing, blockNumber)
	} else {
		watchedBlock, in := fs.dataBlocksMissing[blockNumber]
		if in {
			watchedBlock.probability /= 3
		} else {
			watchedBlock.CID = blockCID
			watchedBlock.probability = 0.33

			// TODO find Peer responsible for this block
			fs.dataBlocksMissing[blockNumber] = watchedBlock
		}

		fs.estimatedBlockProb = (fs.estimatedBlockProb*float32(fs.numBlocks-1) + watchedBlock.probability) / float32(fs.numBlocks)
	}

	// update stats

	// recompute health
}
