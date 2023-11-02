package ipfsconnector

import (
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"

	dag "github.com/ipfs/go-merkledag"
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
}

// TODO:
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

func CreateIPFSGetter(connector *IPFSConnector, CIDIndexMap map[string]int, parityCIDs [][]string, fileCid string, treeCids []string, numBlocks int, emptyTree *EmptyTreeNode, parentMap map[int]int, nodeMap map[int]*EmptyTreeNode) *IPFSGetter {
	indexToDataCIDMap := *util.NewSafeMap()
	indexToDataCIDMap.AddReverseMap(CIDIndexMap)
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
	}
}

// Given an index, first check if this node exists already in the tree,
// it it does, return the data from the node,
// if it doesn't, find the parent of this index, and repeat the procedure,
// we do this until we have an index that either doesn't have parent or whose parent is the same and still can't find its data
func (getter *IPFSGetter) GetData(index int) ([]byte, error) {

	target_node := getter.NodeMap[index]

	// if node contains data just return the data
	if target_node.data != nil {
		return target_node.data, nil
	}

	for {
		// if node doesn't contain data, but the cid exists,
		// then we use the cid to fetch the data from ipfs
		if target_node.CID != "" {
			raw_block, err := getter.GetRawBlock(target_node.CID)
			if err == nil {
				// populate the node with data and links if exists
				dag_node, err := getter.GetDagNodeFromRawBytes(raw_block)
				if err != nil {

					if len(dag_node.Links()) > 0 {
						target_node.data = dag_node.Data()
						for i, dag_child := range dag_node.Links() {
							target_node.Children[i].CID = dag_child.Cid.String()
						}
						return target_node.data, nil
					} else {
						data, err := getter.GetFileDataFromDagNode(dag_node)
						if err != nil {
							return nil, err
						}
						return data, nil
					}

				}

			}
		}

		// if node doesn't contain data and the cid doesn't exist,
		// then we need to find the parent of this node and repeat the procedure
		parent_index, ok := getter.ParentMap[index]
		if !ok || parent_index == index {
			return nil, xerrors.Errorf("no data exists")
		}

		_, err := getter.GetData(parent_index)
		if err != nil {
			return nil, err
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

func (getter *IPFSGetter) GetParityFromTree(index int, strand int) ([]byte, error) {
	if index < 1 || index > getter.NumBlocks {
		err := xerrors.Errorf("invalid index")
		return nil, err
	}
	if strand < 0 || strand > len(getter.TreeCIDs) {
		err := xerrors.Errorf("invalid strand")
		return nil, err
	}

	raw_block, err := getter.GetRawBlock(getter.TreeCIDs[strand])

	if err != nil {
		return nil, err
	}

	dag_node, err := getter.GetDagNodeFromRawBytes(raw_block)
	if err != nil {
		return nil, err
	}

	// TODO: Optimize find the ith leaf!
	// by caching what has been seen so far!
	leaf_curr_count := 0
	var find_ith_leaf func(node *dag.ProtoNode) (*dag.ProtoNode, error)
	find_ith_leaf = func(node *dag.ProtoNode) (*dag.ProtoNode, error) {
		if len(node.Links()) == 0 {
			leaf_curr_count++
			if leaf_curr_count == index {
				return node, nil
			}
		}

		for _, link := range node.Links() {
			raw_child, err := getter.GetRawBlock(link.Cid.String())

			if err != nil {
				return nil, err
			}

			dag_child, err := getter.GetDagNodeFromRawBytes(raw_child)
			if err != nil {
				return nil, err
			}

			leaf, err := find_ith_leaf(dag_child)
			if err != nil {
				return nil, err
			}
			if leaf != nil {
				return leaf, nil
			}
		}

		return nil, nil
	}

	leaf, err := find_ith_leaf(dag_node)

	if err != nil {
		return nil, err
	}

	if leaf == nil {
		return nil, xerrors.Errorf("no parity exists")
	}

	data, err := getter.GetFileDataFromDagNode(leaf)

	return data, err
}
