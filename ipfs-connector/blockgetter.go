package ipfsconnector

import (
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"

	"golang.org/x/xerrors"
)

type IPFSGetter struct {
	entangler.BlockGetter
	*IPFSConnector
	DataIndexCIDMap util.SafeMap
	DataFilter      map[int]struct{}
	Parity          [][]string
	ParityFilter    []map[int]struct{}
	BlockNum        int

	OriginalFileCID string
	TreeCIDs        []string
	NumBlocks       int
	EmptyTree       *EmptyTreeNode
	ParentMap       map[int]int            // maps lattice index of child to lattice index of parent
	NodeMap         map[int]*EmptyTreeNode // maps lattice index of a certain block to its tree node

	ParityTrees     []*ParityTreeNode
	ParityIndexMap  []map[int]*ParityTreeNode
	ParityAvailable []bool

	DataBlocksFetched     int
	DataBlocksCached      int
	DataBlocksUnavailable int
	DataBlocksError       int

	ParityBlocksFetched     int
	ParityBlocksCached      int
	ParityBlocksUnavailable int
	ParityBlocksError       int
}

// 1. save tree depth and max children for parity trees in the metadata
// 2. store this info + other metadata info here
// 3. store empty tree here and use it to guide through the recovery process
// 4. when lattice requests a chunk using the index, we traverse this tree get the path to this chunk
// 5. use the path to recursively get their CIDs and store them here
// 6. use the CIDs to get the data and return it to lattice
// 7. we need to store the parity tree root CIDs here
// 8. when a parity is requested, we traverse the parity tree to get the path to this parity
// 9. use the path to recursively get their CIDs and store them here
// 10. use the CIDs to get the data and return it to lattice
// 11. In case we can't find any of the blocks, should we try with MAX_DEPTH to request the chunks from lattice?
// this would actually make another function call, that would come back here,
// will this keep happening until the root or something else??

func CreateIPFSGetter(connector *IPFSConnector, CIDIndexMap map[string]int, parityCIDs [][]string, fileCid string, treeCids []string, numBlocks int, emptyTree *EmptyTreeNode, parentMap map[int]int, nodeMap map[int]*EmptyTreeNode, parityTrees []*ParityTreeNode, parityIndexMap []map[int]*ParityTreeNode) *IPFSGetter {
	indexToDataCIDMap := *util.NewSafeMap()
	indexToDataCIDMap.AddReverseMap(CIDIndexMap)
	parityAvails := make([]bool, len(parityTrees))
	for i := range parityAvails {
		parityAvails[i] = (parityTrees[i].CID != "")
	}
	return &IPFSGetter{
		IPFSConnector:   connector,
		DataIndexCIDMap: indexToDataCIDMap,
		Parity:          parityCIDs,
		BlockNum:        len(CIDIndexMap),

		// New Fields
		OriginalFileCID: fileCid,
		TreeCIDs:        treeCids,
		NumBlocks:       numBlocks,
		EmptyTree:       emptyTree,
		ParentMap:       parentMap,
		NodeMap:         nodeMap,

		ParityTrees:     parityTrees,
		ParityIndexMap:  parityIndexMap,
		ParityAvailable: parityAvails,

		DataBlocksFetched:       0,
		DataBlocksCached:        0,
		DataBlocksError:         0,
		DataBlocksUnavailable:   0,
		ParityBlocksFetched:     0,
		ParityBlocksCached:      0,
		ParityBlocksUnavailable: 0,
		ParityBlocksError:       0,
	}
}

// Given an index, first check if this node exists already in the tree,
// it it does, return the data from the node,
// if it doesn't, find the parent of this index, and repeat the procedure,
// we do this until we have an index that either doesn't have parent or whose parent is the same and still can't find its data
func (getter *IPFSGetter) GetData(index int) ([]byte, error) {

	//print getter Datafilter

	for k := range getter.DataFilter {
		util.LogPrintf("DataFilter: %d", k)
	}
	/* get the data, mask to represent the data loss */
	if getter.DataFilter != nil {
		if _, ok := getter.DataFilter[index]; ok {
			getter.DataBlocksUnavailable++
			err := xerrors.Errorf("no data exists")
			return nil, err
		}
	}

	util.LogPrintf("Getting data for index %d", index)
	target_node, ok := getter.NodeMap[index]

	if !ok {
		util.LogPrintf("Could not find node for index %d", index)
		getter.DataBlocksError++
		return nil, xerrors.Errorf("no node exists for such index")
	}

	// if node contains data just return the data
	if target_node.Data != nil {
		util.LogPrintf("Found data for index %d", index)
		getter.DataBlocksCached++
		return target_node.Data, nil
	}

	for {
		// if node doesn't contain data, but the cid exists,
		// then we use the cid to fetch the data from ipfs
		if target_node.CID != "" {
			util.LogPrintf("Found CID %s for index %d", target_node.CID, index)
			util.LogPrintf("Attempting to download block using its cid")
			raw_node, err := getter.shell.ObjectGet(target_node.CID)
			if err != nil {
				getter.DataBlocksUnavailable++
				return nil, err
			}
			data, err := getter.GetRawBlock(target_node.CID)
			if err != nil {
				getter.DataBlocksUnavailable++
				return nil, err
			}

			util.LogPrintf("Successfully downloaded block using its cid")
			// populate the node with data and links if exists
			if len(raw_node.Links) > 0 {
				util.LogPrintf("Found %d links for index %d", len(raw_node.Links), index)

				for i, dag_child := range raw_node.Links {
					target_node.Children[i].CID = dag_child.Hash
				}

				util.LogPrintf("Successfully populated node with links")
			}

			getter.DataBlocksFetched++
			target_node.Data = data
			return data, nil

		}

		// if node doesn't contain data and the cid doesn't exist,
		// then we need to find the parent of this node and repeat the procedure
		util.LogPrintf("Could not find cid for index %d, finding its parent", index)
		parent_index, ok := getter.ParentMap[index]
		if !ok || parent_index == index {
			getter.DataBlocksError++
			return nil, xerrors.Errorf("no data exists")
		}

		util.LogPrintf("Found parent for index %d, with index %d", index, parent_index)
		_, err := getter.GetData(parent_index)
		if err != nil {
			getter.DataBlocksUnavailable++
			return nil, err
		}
	}

}

// GetDataCID - mostly redoing of above func with only CID
func (getter *IPFSGetter) GetDataCID(index int) string {
	for k := range getter.DataFilter {
		util.LogPrintf("DataFilter: %d", k)
	}
	/* get the data, mask to represent the data loss */
	if getter.DataFilter != nil {
		if _, ok := getter.DataFilter[index]; ok {
			return ""
		}
	}

	util.LogPrintf("Getting CID for index %d", index)
	target_node, ok := getter.NodeMap[index]

	if !ok {
		util.LogPrintf("Could not find node for index %d", index)
		return ""
	}

	// if node contains CID just return the data
	if target_node.CID != "" {
		util.LogPrintf("Found CID for index %d", index)
		return target_node.CID
	}

	for {
		// if node doesn't contain data, but the cid exists,
		// then we use the cid to fetch the data from ipfs
		if target_node.CID != "" {
			util.LogPrintf("Found CID %s for index %d", target_node.CID, index)
			return target_node.CID

		}

		// if node doesn't contain data and the cid doesn't exist,
		// then we need to find the parent of this node and repeat the procedure
		util.LogPrintf("Could not find CID for index %d, finding its parent", index)
		parent_index, ok := getter.ParentMap[index]
		if !ok || parent_index == index {
			return ""
		}

		util.LogPrintf("Found parent for index %d, with index %d", index, parent_index)
		_, err := getter.GetData(parent_index)
		if err != nil {
			return ""
		}
	}
}

// func (getter *IPFSGetter) GetData(index int) ([]byte, error) {
// 	/* Get the target CID of the block */
// 	cid, ok := getter.DataIndexCIDMap.Get(index)
// 	if !ok {
// 		err := xerrors.Errorf("invalid index")
// 		return nil, err
// 	}

// 	/* get the data, mask to represent the data loss */
// 	if getter.DataFilter != nil {
// 		if _, ok = getter.DataFilter[index]; ok {
// 			err := xerrors.Errorf("no data exists")
// 			return nil, err
// 		}
// 	}
// 	data, err := getter.GetRawBlock(cid)
// 	return data, err

// }

// func (getter *IPFSGetter) GetParity(index int, strand int) ([]byte, error) {
// 	if index < 1 || index > getter.BlockNum {
// 		err := xerrors.Errorf("invalid index")
// 		return nil, err
// 	}
// 	if strand < 0 || strand > len(getter.Parity) {
// 		err := xerrors.Errorf("invalid strand")
// 		return nil, err
// 	}

// 	/* Get the target CID of the block */
// 	cid := getter.Parity[strand][index-1]

// 	/* Get the parity, mask to represent the parity loss */
// 	if getter.ParityFilter != nil && len(getter.ParityFilter) > strand && getter.ParityFilter[strand] != nil {
// 		if _, ok := getter.ParityFilter[strand][index]; ok {
// 			err := xerrors.Errorf("no parity exists")
// 			return nil, err
// 		}
// 	}

// 	data, err := getter.GetFileToMem(cid)
// 	return data, err

// }

// calculateNewBlocks returns the new block indices and the byte ranges
// for a given original block size, new block size, and original index.
func calculateNewBlocks(originalBlockSize, newBlockSize, originalIndex int) [][3]int {
	// Calculate the starting and ending byte for the original block
	startByte := originalIndex * originalBlockSize
	endByte := startByte + originalBlockSize

	// Determine the start and end block numbers in the new block scheme
	startBlock := startByte / newBlockSize
	endBlock := (endByte - 1) / newBlockSize

	// Slice to hold the result ranges
	var result [][3]int

	// If the original block is spread across multiple new blocks
	if startBlock != endBlock {
		// First block range
		result = append(result, [3]int{startBlock, startByte % newBlockSize, newBlockSize})
		// Intermediate full blocks
		for blockIndex := startBlock + 1; blockIndex < endBlock; blockIndex++ {
			result = append(result, [3]int{blockIndex, 0, newBlockSize})
		}
		// Last block range
		result = append(result, [3]int{endBlock, 0, (endByte-1)%newBlockSize + 1})
	} else {
		// The original block is within one new block
		result = append(result, [3]int{startBlock, startByte % newBlockSize, (endByte-1)%newBlockSize + 1})
	}

	return result
}

func (getter *IPFSGetter) GetParityHelper(currentNode *ParityTreeNode, strand int) ([]byte, error) {

	if currentNode == nil {
		getter.ParityBlocksError++
		return nil, xerrors.Errorf("parity doesn't exist")
	}

	// if data already exists just return it
	if currentNode.Data != nil {
		getter.ParityBlocksCached++
		return currentNode.Data, nil
	}

	for {
		// if data doesn't exist, but cid exists, then we use the cid to fetch the data from ipfs
		if currentNode.CID != "" {
			rawNode, err := getter.shell.ObjectGet(currentNode.CID)
			if err != nil {
				getter.ParityBlocksUnavailable++
				return nil, err
			}
			rawBlock, err := getter.GetRawBlock(currentNode.CID)
			if err != nil {
				getter.ParityBlocksUnavailable++
				return nil, err
			}
			dagNode, err := getter.GetDagNodeFromRawBytes(rawBlock)
			if err != nil {
				getter.ParityBlocksUnavailable++
				return nil, err
			}

			// populate the node with data and links if exists
			if len(rawNode.Links) > 0 {
				for i, dag_child := range rawNode.Links {
					currentNode.Children[i].CID = dag_child.Hash
				}
			}

			currentData, err := getter.GetFileDataFromDagNode(dagNode)
			if err != nil {
				getter.ParityBlocksUnavailable++
				return nil, err
			}

			getter.ParityBlocksFetched++
			currentNode.Data = currentData
			return currentData, nil
		}

		// if data doesn't exist and cid doesn't exist, then we need to find the parent of this node and repeat the procedure
		if currentNode.Parent == nil {
			getter.ParityBlocksError++
			return nil, xerrors.Errorf("parity doesn't have a parent")
		}

		_, err := getter.GetParityHelper(currentNode.Parent, strand)

		if err != nil {
			// we only set this to false if a non leaf node can't be found
			// this internal node has no way of being repaired since its not entangled
			// we'll have to make this whole strand unavailable and send a request to
			// the daemon to regenerate and upload the whole strand again
			getter.ParityBlocksUnavailable++
			getter.ParityAvailable[strand] = false
			return nil, err
		}
	}

}

func (getter *IPFSGetter) GetParity(index int, strand int) ([]byte, error) {
	// find the node in ParityIndexMap
	// get the path to the node

	if !getter.ParityAvailable[strand] {
		getter.ParityBlocksUnavailable++
		return nil, xerrors.Errorf("parity tree is missing")
	}

	final_data := make([]byte, 0)
	// TODO: make these variables global and initialize them once!
	blocks := calculateNewBlocks(262158, 262144, index)

	for _, block := range blocks {
		targetNode, ok := getter.ParityIndexMap[strand][block[0]]
		if !ok {
			getter.ParityBlocksError++
			return nil, xerrors.Errorf("no parity exists")
		}
		currentData, err := getter.GetParityHelper(targetNode, strand)

		if err != nil {
			return nil, err
		}

		final_data = append(final_data, currentData[block[1]:block[2]]...)
	}

	return final_data, nil
}

// GetParityCID - return the first CID where the parity block is stored
func (getter *IPFSGetter) GetParityCID(index int, strand int) string {
	if !getter.ParityAvailable[strand] {
		util.LogPrintf("Parity tree is missing: strand=%d", strand)
		return ""
	}

	// TODO: make these variables global and initialize them once!
	blocks := calculateNewBlocks(262158, 262144, index)

	targetNode, ok := getter.ParityIndexMap[strand][blocks[0][0]]
	if !ok {
		util.LogPrintf("No parity exists")
		return ""
	}

	return targetNode.CID
}

// Function is 0-indexed
// function that translates lattice index of a block to its CID
func (getter *IPFSGetter) GetCIDForDataBlock(index int) string {
	if target_node, ok := getter.NodeMap[index]; ok {
		return target_node.CID
	} else {
		return ""
	}
}

// Function is 0-indexed
// function that translates lattice index of a parity block to its CID
func (getter *IPFSGetter) GetCIDForParityBlock(index int, strand int) string {

	if strand < 0 || strand >= len(getter.ParityIndexMap) {
		return ""
	}

	if target_node, ok := getter.ParityIndexMap[strand][index]; !ok {
		return ""
	} else {
		return target_node.CID
	}
}
