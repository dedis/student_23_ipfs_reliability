package Server

import (
	"ipfs-alpha-entanglement-code/entangler"
	"math/rand"
)

const MaxNeighbourTries = 4
const BlockProbThreshold = 0.8 // TEST multiple values
const EstimatorProbLength = 20 // TEST multiple values // estimator reacts at the same speed for small and large files
const HealthSampleSize = 10    // TEST multiple values
const HealthDepth = 2

// ComputeHealth
// @Description: Computes the estimated health of the file, equivalent to its repairability
// HealthSampleSize blocks are sampled at random
func (s *Server) ComputeHealth(fs *FileStats) float32 {
	validCount := 0

	for _, blockNumber := range rand.Perm(len(fs.lattice.DataBlocks))[:HealthSampleSize] {
		_, _, err := fs.lattice.GetChunkDepth(blockNumber+1, HealthDepth)
		if err == nil {
			validCount++
			fs.updateBlockProb(1.0)
			delete(fs.DataBlocksMissing, uint(blockNumber))
		} else {
			blockCID := fs.lattice.Getter.GetDataCID(blockNumber)
			if blockCID != "" {
				s.handleMissingBlock(fs, true, uint(blockNumber), blockCID)
			}
		}
	}

	return (fs.EstimatedBlockProb + float32(validCount)/float32(HealthSampleSize)) / 2
}

func pickNeighbour(fs *FileStats, blockNum *int, isData bool) bool {
	var blocks map[uint]WatchedBlock
	if isData {
		blocks = fs.DataBlocksMissing
	} else {
		blocks = fs.ParityBlocksMissing
	}
	if len(blocks) == 0 {
		return false
	}

	var neighbour *entangler.Block
	found := false

	for i := 0; i < MaxNeighbourTries && !found; i++ {
		n := rand.Intn(len(blocks))

		var block *entangler.Block
		if isData {
			block = fs.lattice.DataBlocks[n]
		} else {
			block = fs.lattice.ParityBlocks[fs.strandNumber][n]
		}

		n = rand.Intn(2)
		if n == 0 {
			if len(block.RightNeighbors) == 0 {
				if len(block.LeftNeighbors) == 0 {
					continue
				}
				neighbour = block.LeftNeighbors[0]
			} else {
				neighbour = block.RightNeighbors[0]
			}
		} else {
			if len(block.LeftNeighbors) == 0 {
				if len(block.RightNeighbors) == 0 {
					continue
				}
				neighbour = block.RightNeighbors[0]
			}
			neighbour = block.LeftNeighbors[0]
		}
		found = true
	}

	if found {
		*blockNum = neighbour.Index
	}

	return found
}

func pickInFailedRegion(badRegions map[string][]string, fs *FileStats, blockNum *int, isData bool) bool {
	var blocks map[uint]WatchedBlock
	if isData {
		blocks = fs.validDataBlocksHistory
	} else {
		blocks = fs.validParityBlocksHistory
	}

	for i, block := range blocks { // TODO map always traversed in the same order ? or can randomize
		_, in := badRegions[block.Peer.Region]
		if in {
			*blockNum = int(i)
			return true
		}
	}

	return false
}

func pickRetry(fs *FileStats, blockNum *int, isData bool) bool {
	var blocks map[uint]WatchedBlock
	if isData {
		blocks = fs.DataBlocksMissing
	} else {
		blocks = fs.ParityBlocksMissing
	}
	if len(blocks) == 0 {
		return false
	}

	n := rand.Intn(len(blocks))

	*blockNum = n
	return true
}

// selectBlockHeuristic
// @Description: Selects a block to inspect based on the current view of the file.
func (s *Server) selectBlockHeuristic(fs *FileStats) (uint, bool) {
	n := rand.Intn(2)
	isData := true
	blockNumber := 0

	if n == 0 {
		// 1/2 chance to select a data block
		n = rand.Intn(8)

		if n < 2 && pickNeighbour(fs, &blockNumber, true) {
			// 1/4 chance to pick a neighbour of a missing block (if exists, else fallback to others random)
		} else if n < 4 && pickInFailedRegion(s.state.potentialFailedRegions, fs, &blockNumber, true) {
			// 1/4 chance to try using tag:region to find new missing blocks (if exists, else fallback to random)
		} else if n < 5 && pickRetry(fs, &blockNumber, true) {
			// 1/8 chance to retry a missing block
		} else {
			// pick random block
			blockNumber = rand.Intn(len(fs.lattice.DataBlocks))
		}

	} else {
		// 1/2 chance to select a parity block
		isData = false
		n = rand.Intn(8)

		if n < 2 && pickNeighbour(fs, &blockNumber, false) {
		} else if n < 4 && pickInFailedRegion(s.state.potentialFailedRegions, fs, &blockNumber, false) {
		} else if n < 5 && pickRetry(fs, &blockNumber, false) {
		} else {
			blockNumber = rand.Intn(len(fs.lattice.ParityBlocks[fs.strandNumber]))
		}
	}

	return uint(blockNumber), isData
}

// InspectFile
// @Description: Inspect a block of the file (parity or data) and update the stats
func (s *Server) InspectFile(fs *FileStats) {

	// select block heuristically
	blockNumber, isData := s.selectBlockHeuristic(fs)

	var blockCID string
	var err error
	// check block
	if isData {
		// fill block CID in lattice
		_, _, err = fs.lattice.GetChunkDepth(int(blockNumber)+1, 1)
		blockCID = fs.lattice.Getter.GetDataCID(int(blockNumber))
	} else {
		// fill block CID in lattice
		_, _, err = fs.lattice.GetParity(int(blockNumber)+1, fs.strandNumber)
		blockCID = fs.lattice.Getter.GetParityCID(int(blockNumber), fs.strandNumber) // TODO verify 1or0 indexed
	}

	if blockCID == "" {
		println("Error: unreachable intermediary node")
		return
	}

	if err == nil {
		fs.updateBlockProb(1.0)
		if isData {
			delete(fs.DataBlocksMissing, blockNumber)
		} else {
			delete(fs.ParityBlocksMissing, blockNumber)
		}
	} else {
		if s.handleMissingBlock(fs, isData, blockNumber, blockCID) {
			return
		}

		if fs.EstimatedBlockProb < BlockProbThreshold {
			fs.Health = s.ComputeHealth(fs)
			if fs.Health < s.repairThreshold {
				// TODO trigger repair (sample values for now)
				s.repairFile(fs, 5, 2) // TODO other values better? depth is important to repair, but numPeers not necessarily as file is not currently accessed (speed less relevant)
				// TODO what if failure to repair with depth 5? try deeper?
			}
		}
	}
}

func (s *Server) handleMissingBlock(fs *FileStats, isData bool, blockNumber uint, blockCID string) bool {
	var watchedBlock WatchedBlock
	var in bool
	if isData {
		watchedBlock, in = fs.DataBlocksMissing[blockNumber]
	} else {
		watchedBlock, in = fs.ParityBlocksMissing[blockNumber]
	}

	if in {
		watchedBlock.Probability /= 3
	} else {
		watchedBlock.CID = blockCID
		watchedBlock.Probability = 0.33

		allocations, err := s.client.IPFSClusterConnector.GetPinAllocations(blockCID) //ips...
		if err != nil {
			return true
		}
		watchedBlock.Peer.Name = allocations[0]

		if watchedBlock.Peer.Name != "" {
			// TODO: fetch region of this peer if possible (from metrics... impl function for this) | get inspired by GetPinAllocations
			watchedBlock.Peer.Region = s.client.IPFSClusterConnector.GetPeerRegionTag(watchedBlock.Peer.Name)

			// watchedBlock.Peer.Region = ...
			if watchedBlock.Peer.Region != "" {
				s.state.potentialFailedRegions[watchedBlock.Peer.Region] = append(s.state.potentialFailedRegions[watchedBlock.Peer.Region], watchedBlock.Peer.Name)
			}
		}

		if isData {
			fs.DataBlocksMissing[blockNumber] = watchedBlock
		} else {
			fs.ParityBlocksMissing[blockNumber] = watchedBlock
		}
	}

	// FIXME: Too harsh?
	fs.updateBlockProb(watchedBlock.Probability)
	return false
}

func (fs *FileStats) updateBlockProb(testedBlockProb float32) {
	// TODO make update with correlation less impactful
	fs.EstimatedBlockProb = (fs.EstimatedBlockProb*float32(EstimatorProbLength-1) + testedBlockProb) / float32(EstimatorProbLength)
}

func (s *Server) repairFile(fs *FileStats, depth uint, numPeers int) {
	op := CollaborativeRepairOperation{
		FileCID:  fs.fileCID,
		MetaCID:  fs.MetadataCID,
		Depth:    depth,
		Origin:   s.address,
		NumPeers: numPeers,
	}

	s.StartCollabRepair(&op)
}
