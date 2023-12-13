package client

import (
	"bytes"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"sync"

	"golang.org/x/xerrors"
)

// Given RootCID of a file and the MetadataCID, regenerate Strand X by downloading data and parity blocks
func (c *Client) RepairStrand(rootCID string, metadataCID string, strand int) (err error) {
	err = c.InitIPFSConnector()
	if err != nil {
		return err
	}

	metaData, err := c.GetMetaData(metadataCID)
	if err != nil {
		return xerrors.Errorf("fail to download metaData: %s", err)
	}

	if (strand < 0) || (strand >= metaData.Alpha) {
		return xerrors.Errorf("invalid strand number")
	}

	// Construct empty tree
	merkleTree, child_parent_index_map, index_node_map, err := ipfsconnector.ConstructTree(metaData.Leaves, metaData.MaxChildren, metaData.Depth, metaData.NumBlocks, metaData.S, metaData.P)

	if err != nil {
		return xerrors.Errorf("fail to construct tree: %s", err)
	}

	merkleTree.CID = metaData.OriginalFileCID

	// for each treeCid, create a new parity tree and indices_map
	parityTrees := make([]*ipfsconnector.ParityTreeNode, len(metaData.TreeCIDs))
	parityIndexMap := make([]map[int]*ipfsconnector.ParityTreeNode, len(metaData.TreeCIDs))

	// Calculate parity tree number of leaves based on the following:
	//TODO: Make these numbers global and initialize them once!
	L_parity := (metaData.NumBlocks*262158 + 262143) / 262144
	K_parity := metaData.MaxParityChildren

	for i, treeCID := range metaData.TreeCIDs {
		curr_tree, curr_map := ipfsconnector.CreateParityTree(L_parity, K_parity)
		parityTrees[i], parityIndexMap[i] = curr_tree, curr_map

		// exclude the cid of the strand we're repairing
		if i != strand {
			parityTrees[i].CID = treeCID
		}

	}

	/* create lattice */
	// create getter
	getter := ipfsconnector.CreateIPFSGetter(c.IPFSConnector, metaData.DataCIDIndexMap, metaData.ParityCIDs, metaData.OriginalFileCID, metaData.TreeCIDs, metaData.NumBlocks, merkleTree, child_parent_index_map, index_node_map, parityTrees, parityIndexMap)

	// We'll set the depth to a larger number since this will be an async process and we can afford to get deeper without affecting
	//  the perceived latency for users
	// create lattice
	lattice := entangler.NewLattice(metaData.Alpha, metaData.S, metaData.P, metaData.NumBlocks, getter, 2)
	lattice.Init()

	option := DownloadOption{
		MetaCID:           metadataCID,
		UploadRecoverData: true,
		DataFilter:        make([]int, 0),
	}

	/* download & recover file from IPFS */
	_, _, errDownload := c.downloadAndRecover(lattice, metaData, option, merkleTree)
	if errDownload != nil {
		return errDownload
	}

	// Regenerate the strand we're repairing

	// start the entangler to read from pipline
	strands := make([]bool, metaData.Alpha)
	for i := 0; i < metaData.Alpha; i++ {
		strands[i] = (i == strand)
	}

	blockNum := metaData.NumBlocks
	dataChan := make(chan []byte, blockNum)
	parityChan := make(chan entangler.EntangledBlock, blockNum)
	tangler := entangler.NewEntangler(metaData.Alpha, metaData.S, metaData.P, strands)

	// start the entangler to read from pipline
	go func() {
		err := tangler.Entangle(dataChan, parityChan)
		if err != nil {
			panic(xerrors.Errorf("could not generate entanglement: %s", err))
		}
	}()

	// send data to entangler
	go func() {
		for i := 1; i <= blockNum; i++ {
			data, _, err := lattice.GetChunk(i)
			if err != nil {
				return
			}
			dataChan <- data
		}
		close(dataChan)
	}()

	parityBlocks := make([][]byte, blockNum)

	// receive entangled blocks from entangler
	var waitGroupAdd sync.WaitGroup
	for block := range parityChan {
		waitGroupAdd.Add(1)
		go func(block entangler.EntangledBlock) {
			defer waitGroupAdd.Done()
			parityBlocks[block.LeftBlockIndex-1] = block.Data
		}(block)
	}
	waitGroupAdd.Wait()

	// Merge entangled blocks into a single byte array
	var mergedParity []byte
	for _, block := range parityBlocks {
		util.LogPrintf("Parity block size: %d", len(block))
		mergedParity = append(mergedParity, block...)
	}

	// Re-upload the whole parity strand
	// We assume that IPFS Cluster would still have the same pinnings
	// and would just redistribute the data
	parityCID, err := c.AddFileFromMem(mergedParity)
	if err != nil {
		return xerrors.Errorf("could not upload parity %d: %s", strand, err)
	}

	if parityCID != metaData.TreeCIDs[strand] {
		return xerrors.Errorf("parity CID mismatch")
	}

	return nil
}

// Function that prepares all data structures needed before downloading or repairing a file

func (c *Client) PrepareRepair(rootCID string, metadataCID string, depth uint) (*Metadata, *ipfsconnector.IPFSGetter, *entangler.Lattice, *ipfsconnector.EmptyTreeNode, error) {

	metaData, err := c.GetMetaData(metadataCID)
	if err != nil {
		return nil, nil, nil, nil, xerrors.Errorf("fail to download metaData: %s", err)
	}

	// Construct empty tree
	merkleTree, child_parent_index_map, index_node_map, err := ipfsconnector.ConstructTree(metaData.Leaves, metaData.MaxChildren, metaData.Depth, metaData.NumBlocks, metaData.S, metaData.P)

	if err != nil {
		return nil, nil, nil, nil, xerrors.Errorf("fail to construct tree: %s", err)
	}

	merkleTree.CID = metaData.OriginalFileCID

	// for each treeCid, create a new parity tree and indices_map
	parityTrees := make([]*ipfsconnector.ParityTreeNode, len(metaData.TreeCIDs))
	parityIndexMap := make([]map[int]*ipfsconnector.ParityTreeNode, len(metaData.TreeCIDs))

	// Calculate parity tree number of leaves based on the following:
	//TODO: Make these numbers global and initialize them once!
	L_parity := (metaData.NumBlocks*262158 + 262143) / 262144
	K_parity := metaData.MaxParityChildren

	for i, treeCID := range metaData.TreeCIDs {
		curr_tree, curr_map := ipfsconnector.CreateParityTree(L_parity, K_parity)
		parityTrees[i], parityIndexMap[i] = curr_tree, curr_map
		parityTrees[i].CID = treeCID
	}

	/* create lattice */
	// create getter
	getter := ipfsconnector.CreateIPFSGetter(c.IPFSConnector, metaData.DataCIDIndexMap, metaData.ParityCIDs, metaData.OriginalFileCID, metaData.TreeCIDs, metaData.NumBlocks, merkleTree, child_parent_index_map, index_node_map, parityTrees, parityIndexMap)

	// create lattice
	lattice := entangler.NewLattice(metaData.Alpha, metaData.S, metaData.P, metaData.NumBlocks, getter, depth)
	lattice.Init()

	return metaData, getter, lattice, merkleTree, nil

}

// Function that repairs all intermediate nodes of a tree
// then for all leaf nodes that aren't available, just reports adds its CID to a list
// Arguments: FileCID, MetaCID, Depth
// returns: List of lattice indices for leaf nodes that need to be repaired

func (c *Client) RetrieveFailedLeaves(rootCID string, metadataCID string, depth uint) (leafIndices []int, err error) {

	_, _, lattice, root, err := c.PrepareRepair(rootCID, metadataCID, depth)
	leafIndices = make([]int, 0)

	if err != nil {
		return nil, err
	}

	// starting from root, traverse the tree and repair all intermediate nodes
	var walker func(*ipfsconnector.EmptyTreeNode) error
	walker = func(node *ipfsconnector.EmptyTreeNode) (err error) {

		// if node is a leaf, check if it's available
		// if not, add it to the list of leaves to be repaired
		if len(node.Children) == 0 {
			_, _, err := lattice.GetChunkDepth(node.LatticeIdx+1, 1)
			if err != nil {
				leafIndices = append(leafIndices, node.LatticeIdx)
			}

			// we don't care about errors here since we already added the CID to the list
			return nil
		}

		// if node is not a leaf, repair it with previously specified depth
		chunk, hasRepaired, err := lattice.GetChunk(node.LatticeIdx + 1)
		if err != nil {
			return xerrors.Errorf("fail to recover chunk with CID: %s", err)
		}

		// upload missing chunk back to the network if allowed
		if hasRepaired {
			// Problem: does trimming zero always works?
			chunk = bytes.Trim(chunk, "\x00")
			err = c.dataReupload(chunk, node.CID, true)
			if err != nil {
				return err
			}
		}

		// unmarshal and iterate
		dagNode, err := c.GetDagNodeFromRawBytes(chunk)
		if err != nil {
			return xerrors.Errorf("fail to parse raw data: %s", err)
		}
		links := dagNode.Links()

		if len(links) != len(node.Children) {
			return xerrors.Errorf("number of links mismatch: %d expected but %d provided", len(node.Children), len(links))
		}

		for i, link := range links {
			node.Children[i].CID = link.Cid.String()
			err = walker(node.Children[i])
			if err != nil {
				return err
			}
		}

		return err
	}

	err = walker(root)

	// TODO: Reupload any parities that were repaired
	// Reupload any parities that were repaired

	return leafIndices, err
}

// Function that takes a list of lattice indices and repairs them
// Arguments: FileCID, MetaCID, Depth, List of indices
// returns: a map of each index to a bool whether it was either repaired(either already available or repaired) or not

func (c *Client) RepairFailedLeaves(rootCID string, metadataCID string, depth uint, leafIndices []int) (result map[int]bool, err error) {

	_, _, lattice, _, err := c.PrepareRepair(rootCID, metadataCID, depth)
	result = make(map[int]bool)
	for _, index := range leafIndices {
		result[index] = false
	}

	if err != nil {
		return result, err
	}

	// for each index, try to get from the lattice and report whether it was retrieved successfully or not
	for _, index := range leafIndices {
		chunk, hasRepaired, err := lattice.GetChunk(index + 1)
		result[index] = (err == nil)
		if hasRepaired {
			// Problem: does trimming zero always works?
			chunk = bytes.Trim(chunk, "\x00")
			e := c.dataReuploadNoCheck(chunk, true)
			result[index] = result[index] && (e == nil)
		}

	}

	// TODO: Reupload any parities that were repaired
	// Reupload any parities that were repaired

	return result, nil
}
