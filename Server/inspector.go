package Server

import (
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"
	"log"
	"math"
	"math/rand"
	"time"
)

const MaxNeighbourTries = 4
const BlockProbThreshold = 0.8 // TEST multiple values
const EstimatorProbLength = 20 // TEST multiple values // estimator reacts at the same speed for small and large files
const HealthSampleSize = 10    // TEST multiple values
const HealthDepth = 2
const RepairDepth = 5
const RepairNumPeers = 2

// ComputeHealth
// @Description: Computes the estimated health of the file, equivalent to its repairability
// HealthSampleSize blocks are sampled at random
func (s *Server) ComputeHealth(fs *FileStats, lattice *entangler.Lattice) float32 {
	validCount := 0

	for _, blockNumber := range rand.Perm(len(lattice.DataBlocks))[:HealthSampleSize] {
		_, _, err := lattice.GetChunkDepth(blockNumber+1, HealthDepth)
		if err == nil {
			validCount++
			fs.updateBlockProb(1.0, false)
			delete(fs.DataBlocksMissing, uint(blockNumber))
		} else {
			blockCID := lattice.Getter.GetDataCID(blockNumber)
			if blockCID != "" {
				s.handleMissingBlock(fs, true, uint(blockNumber), blockCID, false)
			}
		}
	}

	return float32(validCount) / float32(HealthSampleSize)
}

func pickNeighbour(fs *FileStats, blockNum *int, isData bool, lattice *entangler.Lattice) bool {
	var blocks map[uint]*WatchedBlock
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
			block = lattice.DataBlocks[n]
		} else {
			block = lattice.ParityBlocks[fs.strandNumber][n]
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
	var blocks map[uint]*WatchedBlock
	if isData {
		return false // can only know region for blocks which are pinned on IPFS-Cluster (parity blocks are but not data)
	} else {
		blocks = fs.validParityBlocksHistory
	}

	for i, block := range blocks { // Could randomize map traversal
		_, in := badRegions[block.Peer.Region]
		if in {
			*blockNum = int(i)
			return true
		}
	}

	return false
}

func pickRetry(fs *FileStats, blockNum *int, isData bool) bool {
	var blocks map[uint]*WatchedBlock
	if isData {
		blocks = fs.DataBlocksMissing
	} else {
		blocks = fs.ParityBlocksMissing
	}
	if len(blocks) == 0 {
		return false
	}

	n := rand.Intn(len(blocks))

	cnt := 0
	for i, _ := range blocks {
		if cnt == n {
			*blockNum = int(i)
			return true
		}
		cnt++
	}

	return false
}

// selectBlockHeuristic
// @Description: Selects a block to inspect based on the current view of the file.
func (s *Server) selectBlockHeuristic(fs *FileStats, lattice *entangler.Lattice) (uint, bool, bool) {
	n := rand.Intn(2)
	isData := true
	fromInsights := true
	blockNumber := 0

	if n == 0 {
		// 1/2 chance to select a data block
		n = rand.Intn(8)

		if n < 2 && pickNeighbour(fs, &blockNumber, true, lattice) {
			// 1/4 chance to pick a neighbour of a missing block (if exists, else fallback to other methods)
		} else if n < 4 && pickInFailedRegion(s.state.potentialFailedRegions, fs, &blockNumber, true) {
			// 1/4 chance to try using tag:region to find new missing blocks (if exists, else fallback to other methods)
		} else if n < 5 && pickRetry(fs, &blockNumber, true) {
			// 1/8 chance to retry a missing block (if exists, else fallback to other methods)
		} else {
			// pick random block
			blockNumber = rand.Intn(len(lattice.DataBlocks))
			fromInsights = false
		}

	} else {
		// 1/2 chance to select a parity block
		isData = false
		n = rand.Intn(8)
		// same selection as above but for parity blocks
		if n < 2 && pickNeighbour(fs, &blockNumber, false, lattice) {
		} else if n < 4 && pickInFailedRegion(s.state.potentialFailedRegions, fs, &blockNumber, false) {
		} else if n < 5 && pickRetry(fs, &blockNumber, false) {
		} else {
			blockNumber = rand.Intn(len(lattice.ParityBlocks[fs.strandNumber]))
			fromInsights = false
		}
	}

	return uint(blockNumber), isData, fromInsights
}

// InspectFile
// @Description: Inspect a block of the file (parity or data) and update the stats
func (s *Server) InspectFile(fs *FileStats) {

	// get lattice
	_, _, lattice, _, _, err := s.client.PrepareRepair(fs.fileCID, fs.MetadataCID, 2)

	if err != nil {
		println("Error in PrepareRepair: ", err.Error())
		return
	}

	// select block heuristically
	blockNumber, isData, fromInsights := s.selectBlockHeuristic(fs, lattice)

	var blockCID string
	// check block
	if isData {
		blockNumber = uint(math.Min(float64(blockNumber), float64(len(lattice.DataBlocks)-1))) // safeguard
		// fill block CID in lattice
		_, _, err = lattice.GetChunkDepth(int(blockNumber)+1, 1)
		blockCID = lattice.Getter.GetDataCID(int(blockNumber))
	} else {
		blockNumber = uint(math.Min(float64(blockNumber), float64(len(lattice.ParityBlocks[fs.strandNumber])-1))) // safeguard
		// fill block CID in lattice
		_, _, err = lattice.GetParity(int(blockNumber)+1, fs.strandNumber)
		blockCID = lattice.Getter.GetParityCID(int(blockNumber), fs.strandNumber)
	}

	if blockCID == "" {
		if isData {
			s.repairFile(fs)
		} else {
			s.repairStrand(fs)
		}
		println("Error: unreachable intermediary node, repair triggered")
		return
	}

	if err == nil {
		fs.updateBlockProb(1.0, false)
		watchedBlock := WatchedBlock{
			CID:         blockCID,
			Peer:        ClusterPeer{},
			Probability: 1,
		}

		if isData {
			delete(fs.DataBlocksMissing, blockNumber)
		} else {
			_, in := fs.validParityBlocksHistory[blockNumber]
			delete(fs.ParityBlocksMissing, blockNumber)

			if !in || watchedBlock.Peer.Region == "" {
				allocations, err := s.client.IPFSClusterConnector.GetPinAllocations(blockCID)
				if err == nil && len(allocations) > 0 {
					watchedBlock.Peer.Name = allocations[0]

					if watchedBlock.Peer.Name != "" {
						watchedBlock.Peer.Region = s.client.IPFSClusterConnector.GetPeerRegionTag(watchedBlock.Peer.Name)
					}
				}
				fs.validParityBlocksHistory[blockNumber] = &watchedBlock
			}
		}

	} else {
		if s.handleMissingBlock(fs, isData, blockNumber, blockCID, fromInsights) {
			return
		}

		if fs.EstimatedBlockProb < BlockProbThreshold {
			fs.Health = s.ComputeHealth(fs, lattice)
			if fs.Health < s.repairThreshold {
				s.repairFile(fs)
			}
		}
	}
}

func (s *Server) handleMissingBlock(fs *FileStats, isData bool, blockNumber uint, blockCID string, fromInsights bool) bool {
	log.Println("Block[index:", blockNumber, ", CID:", blockCID, "] is missing")
	var watchedBlock *WatchedBlock
	var in bool
	if isData {
		watchedBlock, in = fs.DataBlocksMissing[blockNumber]
	} else {
		watchedBlock, in = fs.ParityBlocksMissing[blockNumber]
	}

	if in {
		watchedBlock.Probability /= 3
	} else {
		watchedBlock = &WatchedBlock{CID: blockCID, Probability: 0.33}

		// register time at which missing block was found
		s.state.unavailableBlocksTimestamps = append(s.state.unavailableBlocksTimestamps, time.Now().UnixNano())

		if isData {
			fs.DataBlocksMissing[blockNumber] = watchedBlock
		} else {
			// parity blocks are pinned => can retrieve region of peer hosting the parity
			allocations, err := s.client.IPFSClusterConnector.GetPinAllocations(blockCID)
			if err != nil || len(allocations) == 0 {
				return true
			}
			watchedBlock.Peer.Name = allocations[0]

			if watchedBlock.Peer.Name != "" {
				watchedBlock.Peer.Region = s.client.IPFSClusterConnector.GetPeerRegionTag(watchedBlock.Peer.Name)

				// watchedBlock.Peer.Region = ...
				if watchedBlock.Peer.Region != "" {
					s.state.potentialFailedRegions[watchedBlock.Peer.Region] = append(s.state.potentialFailedRegions[watchedBlock.Peer.Region], watchedBlock.Peer.Name)
				}
			}
			fs.ParityBlocksMissing[blockNumber] = watchedBlock
		}
	}

	fs.updateBlockProb(watchedBlock.Probability, fromInsights)
	return false
}

func (fs *FileStats) updateBlockProb(testedBlockProb float32, fromInsights bool) {
	if fromInsights {
		// Missing blocks targeted from insights are more likely to be missing
		fs.EstimatedBlockProb = (fs.EstimatedBlockProb*float32(EstimatorProbLength-0.5) + testedBlockProb*0.5) / float32(EstimatorProbLength)
	} else {
		fs.EstimatedBlockProb = (fs.EstimatedBlockProb*float32(EstimatorProbLength-1) + testedBlockProb) / float32(EstimatorProbLength)
	}
}

func (s *Server) repairFile(fs *FileStats) {
	op := CollaborativeRepairOperation{
		FileCID:  fs.fileCID,
		MetaCID:  fs.MetadataCID,
		Depth:    RepairDepth,
		Origin:   s.address,
		NumPeers: RepairNumPeers,
	}

	util.LogPrintf("Repair triggered (data) for file: %s", fs.fileCID)
	s.StartCollabRepair(&op)
}

func (s *Server) repairStrand(fs *FileStats) {
	op := StrandRepairOperation{
		FileCID: fs.fileCID,
		MetaCID: fs.MetadataCID,
		Strand:  fs.strandNumber,
		Depth:   RepairDepth,
	}

	util.LogPrintf("Repair triggered (parity) for file: %s", fs.fileCID)
	s.StartStrandRepair(&op)
}
