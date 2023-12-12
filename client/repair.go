package client

import (
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
