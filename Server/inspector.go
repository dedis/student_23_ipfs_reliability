package Server

import (
	"ipfs-alpha-entanglement-code/entangler"
)

// ComputeHealth
// @Description: Computes the estimated health of the file, corresponding to repairability of the file.
// This value is based on the missing blocks and the current state of the cluster.
func (fs *FileStats) ComputeHealth() float32 {
	// TODO: Update (repairability based on lattice with missing blocks and estimated block probability)
	// sample three diff lattices with the missing blocks and the estimated block probability
	// then compute the average repairability of the file (using the max repair depth of each)

	// TODO new health definition: how many blocks are missing with depth 2 repair

	return fs.EstimatedBlockProb
}

// selectBlockHeuristic
// @Description: Selects a block to inspect based on the current view of the file.
func (s *Server) selectBlockHeuristic(fs *FileStats, lattice *entangler.Lattice) (uint, bool) {
	// TODO: Make usage of missing blocks and cluster state (region, etc.)
	//  principally seek neighbours of missing blocks (in the lattice)

	// 1/2 chance to select a data block
	// 		1/5 chance to retry a missing block
	//      1/5 chance to select a block at random
	//      1/5 chance to try using tag:region to find new missing blocks (if exists, else fallback to random)
	//      2/5 chance to pick a neighbour of a missing block (if exists, else fallback to random)
	//
	// 1/2 chance to select a parity block
	//      same...

	return uint(len(lattice.DataBlocks)), true
}

// InspectFile
// @Description: Inspect a block of the file (parity or data) and update the stats
func (s *Server) InspectFile(fileCID string, fs *FileStats) {

	_, _, lattice, _, _, err := s.client.PrepareRepair(fileCID, fs.StrandRootCID, 2)
	// TODO is 5th return value required? = indexNodeMap (replaced it with Getter.GetData/ParityCID)

	if err != nil {
		println("Error in PrepareRepair: ", err.Error())
		return
	}

	// select block heuristically
	blockNumber, isData := s.selectBlockHeuristic(fs, lattice)

	var blockCID string
	// check block
	if isData {
		_, _, err = lattice.GetChunkDepth(int(blockNumber)+1, 1)
		blockCID = lattice.Getter.GetDataCID(int(blockNumber)) // FIXME: How is it updated by getchunkdepth? (*indexNodeMap)[int(blockNumber)].CID
	} else {
		strandNumber := getStrandNumber(lattice)
		_, _, err = lattice.GetParity(int(blockNumber)+1, strandNumber)
		blockCID = lattice.Getter.GetParityCID(int(blockNumber), strandNumber) // TODO verify 1or0 indexed
	}

	if blockCID == "" {
		println("Error: unreachable intermediary node")
		return
	}

	totalNbBlocks := len(lattice.DataBlocks) + len(lattice.ParityBlocks[0])

	if err == nil {
		fs.updateBlockProb(1.0, totalNbBlocks)
		delete(fs.DataBlocksMissing, blockNumber)
	} else {
		watchedBlock, in := fs.DataBlocksMissing[blockNumber]
		if in {
			watchedBlock.Probability /= 3
		} else {
			watchedBlock.CID = blockCID
			watchedBlock.Probability = 0.33

			allocations, err := s.client.IPFSClusterConnector.GetPinAllocations(blockCID) //ips... without port
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
		fs.updateBlockProb(watchedBlock.Probability, totalNbBlocks)

		fs.Health = fs.ComputeHealth()
		if fs.Health < s.repairThreshold {
			// TODO trigger repair
		}
	}
}

func (fs *FileStats) updateBlockProb(testedBlockProb float32, totalNbBlocks int) {
	fs.EstimatedBlockProb = (fs.EstimatedBlockProb*float32(totalNbBlocks-1) + testedBlockProb) / float32(totalNbBlocks)
}

func getStrandNumber(lattice *entangler.Lattice) int {
	i := 0
	for ; !lattice.Strands[i]; i++ {
	}
	return i
}
