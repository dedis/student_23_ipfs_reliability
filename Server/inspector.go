package Server

import "math/rand"

func (fs *FileStats) Health() float32 {
	// TODO: Update (repairability based on lattice with missing blocks and estimated block probability)
	return fs.estimatedBlockProb
}

// InspectFile
// @Description: Inspect a block of the file (parity or data) and update the stats
func (s *Server) InspectFile(fileCID string, fs *FileStats) {
	// TODO: impl

	// select block heuristically
	blockNumber := uint(rand.Intn(fs.numBlocks))
	// TODO get CID from Block number
	blockCID := fileCID // FIXME <-

	// check block
	_, err := s.sh.BlockGet(blockCID)
	// FIXME check if timeout tell: block was not there
	//  can store time to download somewhere and ser a timeout appropriately...

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
