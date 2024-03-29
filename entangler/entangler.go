package entangler

import (
	"ipfs-alpha-entanglement-code/util"
	"os"

	"golang.org/x/xerrors"
)

// Position defines which position category the original block belongs
type PositionClass int8

const (
	Top PositionClass = iota
	Central
	Bottom
)

// Strand defines which strand the entangled block belongs
type StrandClass int

const (
	Horizontal StrandClass = iota
	Right
	Left
)

// EntangledBlock is the parity block output by alpha entanglement code
type EntangledBlock struct {
	LeftBlockIndex  int
	RightBlockIndex int
	Data            []byte
	Strand          StrandClass
}

// NewEntangledBlock creates a new entangled block and set it strandclass according to the input
func NewEntangledBlock(l int, r int, data []byte, strand int) (block *EntangledBlock) {
	block = &EntangledBlock{LeftBlockIndex: l, RightBlockIndex: r, Data: data, Strand: StrandClass(strand)}

	return block
}

// Entangler manages all the entanglement related behaviors
type Entangler struct {
	Alpha    int // now only support alpha = 3
	S        int
	P        int
	ChunkNum int

	ChainStartData       [][]byte
	MaxChainNumPerStrand int

	// cached data. reset for each entanglement
	cachedParities     [][]*EntangledBlock
	parityBlocksToWrap [][]*EntangledBlock

	// Strands to generate
	// For each index, specifies whether the strand should be generated
	Strands []bool
}

// NewEntangler takes the entanglement paramters and the original data slice and creates an entangler
func NewEntangler(alpha int, s int, p int, strands []bool) (entangler *Entangler) {
	// value check. See details in alpha-entanglement-code paper (https://ieeexplore.ieee.org/document/8416482)
	if alpha < 1 {
		util.ThrowError("invalid value. Expect alpha > 0")
	}
	if alpha == 1 && !(s == 1 && p == 0) {
		util.ThrowError("invalid value. Expect s = 1 and p = 0")
	}
	if alpha > 1 && s > p {
		util.ThrowError("invalid value. Expect p >= s")
	}

	entangler = &Entangler{Alpha: alpha, S: s, P: p}
	if s > p {
		entangler.MaxChainNumPerStrand = s
	} else {
		entangler.MaxChainNumPerStrand = p
	}

	if len(strands) == 0 {
		for i := 0; i < alpha; i++ {
			entangler.Strands = append(entangler.Strands, true)
		}
	} else {
		entangler.Strands = strands
	}

	return entangler
}

// WriteEntanglementToFile writes the entanglement into files
func (e *Entangler) WriteEntanglementToFile(chunkSize int, path []string, parityChan chan EntangledBlock) (err error) {
	if len(path) != e.Alpha {
		err = xerrors.Errorf("Invalid number of entanglement output paths. %d expected but %d provided", e.Alpha, len(path))
		return err
	}

	parities := make([][][]byte, e.Alpha)
	for k := 0; k < e.Alpha; k++ {
		if !e.Strands[k] {
			continue
		}
		parities[k] = make([][]byte, e.ChunkNum)
	}
	for parity := range parityChan {
		util.InfoPrintf(util.Yellow("Strand %d: (%d, %d)\n"), parity.Strand, parity.LeftBlockIndex, parity.RightBlockIndex)
		if !e.Strands[parity.Strand] {
			continue
		}
		parities[parity.Strand][parity.LeftBlockIndex-1] = parity.Data
	}

	for k := 0; k < e.Alpha; k++ {
		if !e.Strands[k] {
			continue
		}
		// generate byte array of the current strand
		entangledData := make([]byte, 0)
		for _, parityData := range parities[k] {
			if chunkSize > 0 {
				c := make([]byte, chunkSize)
				copy(c, parityData)
				entangledData = append(entangledData, c...)
			} else {
				entangledData = append(entangledData, parityData...)
			}
			entangledData = append(entangledData, parityData...)
		}

		// write entanglement to file
		err = os.WriteFile(path[k], entangledData, 0600)
		if err != nil {
			return err
		}
	}

	return err
}

// Entangle generate the entangelement for the given arrray of blocks
func (e *Entangler) Entangle(dataChan chan []byte, parityChan chan EntangledBlock) error {
	e.prepareEntangle()

	// generate the lattice
	util.LogPrintf("Start generating lattice")
	index := 0
	for block := range dataChan {
		index++
		e.entangleSingleBlock(index, block, parityChan)
		if index <= e.MaxChainNumPerStrand {
			e.ChainStartData[index-1] = block
		}
	}
	e.ChunkNum = index
	util.LogPrintf("Finish generating lattice")

	// wraps the lattice
	util.LogPrintf("Start wrapping lattice")
	e.wrapLattice(parityChan)
	util.LogPrintf("Finish wrapping lattice")

	close(parityChan)

	return nil
}

// prepareEntangle prepares the data structure that will be used for entanglement
func (e *Entangler) prepareEntangle() {
	e.parityBlocksToWrap = make([][]*EntangledBlock, e.Alpha)
	for k := 0; k < e.Alpha; k++ {
		if !e.Strands[k] {
			continue
		}
		e.parityBlocksToWrap[k] = make([]*EntangledBlock, e.MaxChainNumPerStrand)
	}

	e.ChainStartData = make([][]byte, e.MaxChainNumPerStrand)

	e.cachedParities = make([][]*EntangledBlock, e.Alpha)

	if e.Strands[0] {
		e.cachedParities[0] = make([]*EntangledBlock, e.S)
		for i := 0; i < e.S; i++ {
			e.cachedParities[0][i] = NewEntangledBlock(0, 0, make([]byte, 0), 0)
		}
	}

	for k := 1; k < e.Alpha; k++ {
		if !e.Strands[k] {
			continue
		}

		e.cachedParities[k] = make([]*EntangledBlock, e.P)
		for i := 0; i < e.P; i++ {
			e.cachedParities[k][i] = NewEntangledBlock(0, 0, make([]byte, 0), k)
		}
	}
}

// entangleSingleBlock reads the backward parity neighbors from cache
// and produce the corresponding forward parity neighbors. It should be
// called in the correct order to ensure the correctness of cached blocks
func (e *Entangler) entangleSingleBlock(index int, data []byte, parityChan chan EntangledBlock) {
	cachePos := e.getChainIndexes(index)
	rIndexes := e.getForwardNeighborIndexes(index)

	for k := 0; k < e.Alpha; k++ {
		if !e.Strands[k] {
			continue
		}

		// read parity block from cache
		prevBlock := e.cachedParities[k][cachePos[k]]
		// generate new parity block
		parityData := xorChunkData(data, prevBlock.Data)
		// generate, cache and store entangled block
		nextBlock := NewEntangledBlock(index, rIndexes[k], parityData, k)
		e.cachedParities[k][cachePos[k]] = nextBlock
		if e.getChainStartIndexes(index)[k] != index {
			parityChan <- *nextBlock
		} else {
			e.parityBlocksToWrap[k][index-1] = nextBlock
		}
	}
}

// wrapLattice wraps the lattice by modify the first parities on each strand
func (e *Entangler) wrapLattice(parityChan chan EntangledBlock) {
	for k, cacheParity := range e.cachedParities {
		if !e.Strands[k] {
			continue
		}
		for _, parityNode := range cacheParity {
			// Link the last parity block to the first data block of the chain
			index := e.getChainStartIndexes(parityNode.RightBlockIndex)[k]
			parityNode.RightBlockIndex = index
			// Recompute the first parity block
			rIndex := e.getForwardNeighborIndexes(index)[k]
			if e.IsValidIndex(rIndex) {
				// the first block is not the rightmost block
				rNext := NewEntangledBlock(index, rIndex,
					xorChunkData(e.ChainStartData[index-1], parityNode.Data), k)
				e.parityBlocksToWrap[k][index-1] = rNext
			}
			parityChan <- *e.parityBlocksToWrap[k][index-1]
		}
	}
}

// getPositionCategory determines which category the node belongs. Top, Bottom or Central
func (e *Entangler) getPositionCategory(index int) PositionClass {
	nodePos := index % e.S
	if nodePos == 1 || nodePos == 1-e.S {
		return Top
	} else if nodePos == 0 {
		return Bottom
	}
	return Central
}

// getChainIndexes reads the cached backward parity neighbors of the current indexed node
func (e *Entangler) getChainIndexes(index int) (indexes []int) {
	h := (index - 1) % e.S

	indexInWindow := (index - 1) % (e.S * e.P)
	x := indexInWindow % e.P
	y := indexInWindow / e.P

	rh := (y - x + e.P) % e.P
	lh := (x + y) % e.S

	indexes = []int{h, rh, lh}

	return indexes
}

// getChainStartIndexes returns the position of the first node on the chain where the indexed node is on
func (e *Entangler) getChainStartIndexes(index int) (indexes []int) {
	indexes = e.getChainIndexes(index)
	indexes[0]++
	indexes[1] = (e.P-indexes[1])%e.P + 1
	indexes[2]++

	return indexes
}

// getBackwardNeighborIndexes returns the index of backward neighbors that can be entangled with current node
// See details in alpha-entanglement-code paper (https://ieeexplore.ieee.org/document/8416482)
// func (e *Entangler) getBackwardNeighborIndexes(index int) (indexes []int) {
// 	if e.Alpha > 3 {
// 		util.ThrowError("alpha should equal 3")
// 	}

// 	// d_i is tangled with p_{h,i}
// 	pos := e.getPositionCategory(index)
// 	var h, rh, lh int
// 	switch pos {
// 	case Top:
// 		h = index - e.S
// 		rh = index - e.S*e.P + (e.S*e.S - 1)
// 		lh = index - (e.S - 1)
// 	case Central:
// 		h = index - e.S
// 		rh = index - (e.S + 1)
// 		lh = index - (e.S - 1)
// 	case Bottom:
// 		h = index - e.S
// 		rh = index - (e.S + 1)
// 		lh = index - e.S*e.P + (e.S-1)*(e.S-1)
// 	}

// 	indexes = []int{h, rh, lh}

// 	return indexes
// }

// getForwardNeighborIndexes returns the index of forward neighbors that is the entangled output of current node
// See details in alpha-entanglement-code paper (https://ieeexplore.ieee.org/document/8416482)
func (e *Entangler) getForwardNeighborIndexes(index int) (indexes []int) {
	if e.Alpha > 3 {
		util.ThrowError("alpha should equal 3")
	}

	// d_i creates entangled block p_{i,j}
	pos := e.getPositionCategory(index)
	var h, rh, lh int
	switch pos {
	case Top:
		h = index + e.S
		rh = index + e.S + 1
		lh = index + e.S*e.P - (e.S-1)*(e.S-1)
	case Central:
		h = index + e.S
		rh = index + e.S + 1
		lh = index + e.S - 1
	case Bottom:
		h = index + e.S
		rh = index + e.S*e.P - (e.S*e.S - 1)
		lh = index + e.S - 1
	}

	indexes = []int{h, rh, lh}

	return indexes
}

// IsValidIndex checks if the index is inside the lattice
func (e *Entangler) IsValidIndex(index int) bool {
	if index < 1 || index > e.ChunkNum {
		return false
	}
	return true
}
