package client

import (
	"bytes"
	"encoding/json"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/util"
	"log"
	"net/http"
	"sync"

	"golang.org/x/xerrors"
)

// Upload uploads the original file, generates and uploads the entanglement of that file
func (c *Client) Upload(path string, alpha int, s int, p int, replicationFactor int, communityNodeAddress string) (rootCID string,
	metaCID string, pinResult func() error, err error) {

	// // init ipfs connector. Fail the whole process if no connection built
	// err = c.InitIPFSConnector()
	// if err != nil {
	// 	return "", "", nil, err
	// }

	/* add original file to ipfs */

	peers_ := c.IPFSClusterConnector.GetAllPeers()

	util.LogPrintf("Total %d peers in cluster", len(peers_))
	// print all peer information
	for id, peer := range peers_ {
		util.LogPrintf("Peer %s: %s", id, peer)
	}

	rootCID, err = c.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	util.LogPrintf("Finish adding file to IPFS with CID %s. File path: %s", rootCID, path)
	if alpha < 1 {
		// expect no entanglement
		return rootCID, "", nil, nil
	}

	/* get merkle tree from IPFS and flatten the tree */

	root, maxChildren, maxDepth, err := c.GetMerkleTree(rootCID, &entangler.Lattice{})
	if err != nil {
		return rootCID, "", nil, xerrors.Errorf("could not read merkle tree: %s", err)
	}
	nodes := root.GetFlattenedTree(s, p, true)
	blockNum := len(nodes)
	leaves := 0
	util.LogPrintf(util.Green("Number of nodes in the merkle tree is %d. Node sequence:"), blockNum)
	for idx, node := range nodes {
		util.LogPrintf(util.Green(" %d"), node.PreOrderIdx)
		data, err := node.Data()

		if len(node.Children) == 0 {
			leaves++
		}

		if err != nil {
			log.Fatal(err)
		}
		util.LogPrintf("Data size: %d", len(data))
		util.LogPrintf("Child number: %d", len(node.Children))
		util.LogPrintf("Node lattice Index: %d, preorder index: %d", idx, node.PreOrderIdx)
	}
	util.LogPrintf("\n")
	util.LogPrintf("Finish reading and flattening file's merkle tree from IPFS")

	/* generate entanglement */

	parityCIDs, parityBlocks, err := c.generateEntanglementAndUpload(alpha, s, p, nodes)
	if err != nil {
		return rootCID, "", nil, err
	}

	// // init cluster connector. Delay th fail after all uploading to IPFS finishes
	// clusterErr := c.InitIPFSClusterConnector()
	// if clusterErr != nil {
	// 	return rootCID, metaCID, nil, clusterErr
	// }

	/* pin files in cluster */
	treeCids, maxParityChildren, err := c.pinAlphaEntanglements(alpha, parityBlocks, replicationFactor)
	if err != nil {
		return rootCID, "", nil, err
	}

	/* Store Metatdata */

	cidMap := make(map[string]int)
	for i, node := range nodes {
		cidMap[node.CID] = i + 1
	}
	metaData := Metadata{
		Alpha:           alpha,
		S:               s,
		P:               p,
		RootCID:         rootCID,
		DataCIDIndexMap: cidMap,
		ParityCIDs:      parityCIDs,
		//new fields
		NumBlocks:       len(nodes), // N
		OriginalFileCID: rootCID,
		TreeCIDs:        treeCids,
		MaxChildren:     maxChildren, // K
		Leaves:          leaves,      // L
		Depth:           maxDepth,    // D

		MaxParityChildren: maxParityChildren,
	}
	rawMetadata, err := json.Marshal(metaData)
	if err != nil {
		return rootCID, "", nil, xerrors.Errorf("could not marshal metadata: %s", err)
	}
	metaCID, err = c.AddFileFromMem(rawMetadata)
	if err != nil {
		return rootCID, "", nil, xerrors.Errorf("could not upload metadata: %s", err)
	}
	util.LogPrintf("File CID: %s. MetaFile CID: %s", rootCID, metaCID)

	pinResult = c.pinMetadataAndParities(metaCID, parityCIDs)

	// Notify IPFS-Community Node that ROOT CIDs must be tracked (if requested)
	if communityNodeAddress != "" { // TODO: Confirm that community node is up?
		log.Println("Trying to start tracking for rootCIDs")
		requestStruct := ForwardMonitoringRequest{
			FileCID:        rootCID,
			MetadataCID:    metaCID,
			StrandRootCIDs: treeCids,
		}

		requestPayload, err := json.Marshal(requestStruct)
		if err != nil {
			log.Println("(error creating request) Couldn't start tracking for : ", rootCID)
			return rootCID, metaCID, pinResult, nil
		}

		// Send the POST request
		req, err := http.NewRequest("POST", "http://"+communityNodeAddress+"/forwardMonitoring", bytes.NewBuffer(requestPayload))
		if err != nil {
			log.Println("(error creating http request) Couldn't start tracking for : ", rootCID)
			return rootCID, metaCID, pinResult, nil
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			log.Println("(error sending request) Couldn't start tracking for : ", rootCID)
			return rootCID, metaCID, pinResult, nil
		} else {
			log.Println("Request sent for monitoring on address: ", communityNodeAddress+"/forwardMonitoring")
		}
		defer resp.Body.Close()
	}

	return rootCID, metaCID, pinResult, nil
}

// generateLattice takes a slice of flattened tree as well as alpha, s, p to perform alpha entanglement
func (c *Client) generateEntanglementAndUpload(alpha int, s int, p int,
	nodes []*ipfsconnector.TreeNode) ([][]string, [][][]byte, error) {

	blockNum := len(nodes)
	dataChan := make(chan []byte, blockNum)
	parityChan := make(chan entangler.EntangledBlock, alpha*blockNum)

	// start the entangler to read from pipline
	tangler := entangler.NewEntangler(alpha, s, p, []bool{})
	go func() {
		err := tangler.Entangle(dataChan, parityChan)
		if err != nil {
			panic(xerrors.Errorf("could not generate entanglement: %s", err))
		}
	}()

	// send data to entangler
	go func() {
		for _, node := range nodes {
			nodeData, err := node.Data()
			if err != nil {
				return
			}
			dataChan <- nodeData
		}
		close(dataChan)
	}()

	/* store parity blocks one by one */

	parityCIDs := make([][]string, alpha)
	for k := 0; k < alpha; k++ {
		parityCIDs[k] = make([]string, blockNum)
	}

	parityBlocks := make([][][]byte, alpha)
	for k := 0; k < alpha; k++ {
		parityBlocks[k] = make([][]byte, blockNum)
	}

	var waitGroupAdd sync.WaitGroup
	for block := range parityChan {
		waitGroupAdd.Add(1)

		go func(block entangler.EntangledBlock) {
			defer waitGroupAdd.Done()
			parityBlocks[block.Strand][block.LeftBlockIndex-1] = block.Data

			// upload file to IPFS network
			blockCID, err := c.AddFileFromMem(block.Data)
			if err == nil {
				parityCIDs[block.Strand][block.LeftBlockIndex-1] = blockCID
			}
		}(block)
	}
	waitGroupAdd.Wait()

	// check if all parity blocks are added successfully
	for k := 0; k < alpha; k++ {
		util.LogPrintf("Displaying parities for strand %d", k)
		for i, parity := range parityCIDs[k] {
			util.LogPrintf("Parity %d: %s", i, parity)
			if len(parity) == 0 {
				return nil, nil, xerrors.Errorf("could not upload parity %d on strand %d\n", i, k)
			}
		}
		util.LogPrintf("Finish uploading entanglement %d", k)
	}

	// c.pinAlphaEntanglements(alpha, parityBlocks)

	return parityCIDs, parityBlocks, nil
}

func (c *Client) pinAlphaEntanglements(alpha int, parityBlocks [][][]byte, replicationFactor int) ([]string, int, error) {
	// for each strand, merge all bytes into one byte array
	// then upload the whole file to IPFS
	// retreieve the merkle tree of each file
	// for each tree, recursively pin the root node and all its children
	// finally return each of the strand's root node's CID

	// TODO: Try to find if there's another way for this!
	// we will define each block so that it would be exactly 2 * 256 KB (which is the maximum for each block in IPFS)
	// targetSize := 2 * 262144

	currentMaxChildren := 0
	parityCIDs := make([]string, alpha)
	for k := 0; k < alpha; k++ {
		// merge all bytes into one byte array
		var mergedParity []byte
		util.LogPrintf("Merging entanglement %d", k)
		for _, block := range parityBlocks[k] {
			util.LogPrintf("Parity block size: %d", len(block))
			// padding := make([]byte, targetSize-len(block))
			mergedParity = append(mergedParity, block...)
			// mergedParity = append(mergedParity, padding...)
		}

		// print mergedParity size
		util.LogPrintf("Merged parity size: %d", len(mergedParity))
		// upload file to IPFS network
		blockCID, err := c.AddFileFromMem(mergedParity)
		if err != nil {
			return nil, 0, xerrors.Errorf("could not upload parity %d: %s", k, err)
		}

		util.LogPrintf("Finish uploading entanglement %d with root cid %s", k, blockCID)

		// pin the whole file block by block
		tmpMaxChildren, err := c.pinEntanglementTree(blockCID, replicationFactor)
		if err != nil {
			return nil, 0, xerrors.Errorf("could not pin parity %d: %s", k, err)
		}

		if tmpMaxChildren > currentMaxChildren {
			currentMaxChildren = tmpMaxChildren
		}
		util.LogPrintf("Finish uploading entanglement %d with root cid %s", k, blockCID)
		parityCIDs[k] = blockCID
	}

	return parityCIDs, currentMaxChildren, nil
}

func (c *Client) pinEntanglementTree(entaglementCID string, replicationFactor int) (int, error) {
	// get the merkle tree from IPFS
	currentMaxChildren := 0
	tree, _, _, err := c.GetMerkleTree(entaglementCID, nil)
	if err != nil {
		return 0, xerrors.Errorf("could not get merkle tree: %s", err)
	}
	// Strand 0 , parity 0
	//Qma3dPFfrYjS8yGbyMZCrrRNDq6oFKhykmqEnuSnAvhp85

	// recursively pin the root node and all its children
	var walker func(*ipfsconnector.TreeNode)
	walker = func(parent *ipfsconnector.TreeNode) {
		if parent == nil {
			return
		}
		util.LogPrintf("pinning node %s", parent.CID)
		// pin the current node
		// if leaf then just pin once, otherwise pin replicationFactor times
		var err error
		if len(parent.Children) == 0 {
			err = c.IPFSClusterConnector.AddPinDirect(parent.CID, 1)
		} else {
			err = c.IPFSClusterConnector.AddPinDirect(parent.CID, replicationFactor)
		}
		if err != nil {
			log.Printf("could not pin node %s: %s", parent.CID, err)
			return
		}
		if len(parent.Children) > currentMaxChildren {
			currentMaxChildren = len(parent.Children)
		}
		if len(parent.Children) > 0 {
			for _, child := range parent.Children {
				walker(child)
			}
		}
	}

	walker(tree)

	return currentMaxChildren, nil
}

// pinMetadataAndParities pins the metadata and parities in IPFS cluster in the non-blocking way
// User could use the returned function to wait and check if there is any error
func (c *Client) pinMetadataAndParities(metaCID string, parityCIDs [][]string) func() error {
	var waitGroupPin sync.WaitGroup
	waitGroupPin.Add(1)
	var PinErr error
	go func() {
		defer waitGroupPin.Done()

		err := c.IPFSClusterConnector.AddPin(metaCID, 0)
		if err != nil {
			PinErr = xerrors.Errorf("could not pin metadata: %s", err)
			return
		}

		for i := 0; i < len(parityCIDs); i++ {
			for j := 0; j < len(parityCIDs[0]); j++ {
				err := c.IPFSClusterConnector.AddPin(parityCIDs[i][j], 1)
				if err != nil {
					PinErr = xerrors.Errorf("could not pin parity %s: %s", parityCIDs[i][j], err)
					return
				}
			}
		}
	}()

	pinResult := func() (err error) {
		waitGroupPin.Wait()
		return PinErr
	}

	return pinResult
}
