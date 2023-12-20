package Server

import (
	"ipfs-alpha-entanglement-code/entangler"
	"math/rand"
)

const MaxNeighbourTries = 4

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
		if n == 0 { // TODO simplify if-logic
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
		n = rand.Intn(7)

		if n < 3 && pickNeighbour(fs, &blockNumber, true) {
			// 3/7 chance to pick a neighbour of a missing block (if exists, else fallback to others random)
		} else if n < 5 && pickInFailedRegion(s.state.potentialFailedRegions, fs, &blockNumber, true) {
			// 2/7 chance to try using tag:region to find new missing blocks (if exists, else fallback to random)
		} else if n < 6 && pickRetry(fs, &blockNumber, true) {
			// 1/7 chance to retry a missing block
		} else {
			// pick random block
			blockNumber = rand.Intn(len(fs.lattice.DataBlocks))
		}

	} else {
		// 1/2 chance to select a parity block
		isData = false
		n = rand.Intn(7)

		if n < 3 && pickNeighbour(fs, &blockNumber, false) {
		} else if n < 5 && pickInFailedRegion(s.state.potentialFailedRegions, fs, &blockNumber, false) {
		} else if n < 6 && pickRetry(fs, &blockNumber, false) {
		} else {
			blockNumber = rand.Intn(len(fs.lattice.ParityBlocks[fs.strandNumber]))
		}
	}

	return uint(len(fs.lattice.DataBlocks)), isData
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

	totalNbBlocks := len(fs.lattice.DataBlocks) + len(fs.lattice.ParityBlocks[0])

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

			allocations, err := s.client.IPFSClusterConnector.GetPinAllocations(blockCID) //ips...
			if err != nil {
				return
			}
			watchedBlock.Peer.Name = allocations[0]
			// TODO: fetch region of this peer if possible (from metrics... impl function for this) | get inspired by GetPinAllocations
			// watchedBlock.Peer.Region := s.client.IPFSClusterConnector.GetPeerRegionTag(watchedBlock.Peer.Name)

			// watchedBlock.Peer.Region = ...
			if watchedBlock.Peer.Region != "" {
				s.state.potentialFailedRegions[watchedBlock.Peer.Region] = append(s.state.potentialFailedRegions[watchedBlock.Peer.Region], watchedBlock.Peer.Name)
			}

			if isData {
				fs.DataBlocksMissing[blockNumber] = watchedBlock
			} else {
				fs.ParityBlocksMissing[blockNumber] = watchedBlock
			}
		}

		// FIXME: Too harsh?
		fs.updateBlockProb(watchedBlock.Probability, totalNbBlocks)

		fs.Health = fs.ComputeHealth()
		if fs.Health < s.repairThreshold {
			// TODO trigger repair (sample func for now)
			s.repairFile(fs, 4, 2)
		}
	}
}

func (fs *FileStats) updateBlockProb(testedBlockProb float32, totalNbBlocks int) {
	fs.EstimatedBlockProb = (fs.EstimatedBlockProb*float32(totalNbBlocks-1) + testedBlockProb) / float32(totalNbBlocks)
}

func (s *Server) repairFile(fs *FileStats, depth uint, numPeers int) {
	op := CollaborativeRepairOperation{
		FileCID:  fs.fileCID,
		MetaCID:  fs.MetadataCID,
		Depth:    depth,
		Origin:   s.clusterIP, // TODO In the preventive repair case, there is no need to forward the result to an "origin peer". Blocks should simply be re-updated
		NumPeers: numPeers,
	}

	s.StartCollabRepair(&op)
}
