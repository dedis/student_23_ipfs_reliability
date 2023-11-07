package ipfsconnector

import (
	"errors"
	"fmt"
)

type ParityTreeNode struct {
	Data     []byte
	Children []*ParityTreeNode
	Parent   *ParityTreeNode
	Index    int
	CID      string
}

type EmptyTreeNode struct {
	Data        []byte
	Children    []*EmptyTreeNode
	Parent      *EmptyTreeNode
	Depth       int
	PreOrderIdx int
	LatticeIdx  int
	CID         string
}

// TreeNode implements a node in IPLD Merkle Tree
type TreeNode struct {
	data []byte

	Children    []*TreeNode
	Parent      *TreeNode
	Depth       int
	TreeSize    int
	LeafSize    int
	PreOrderIdx int
	LatticeIdx  int

	connector *IPFSConnector
	CID       string
}

// CreateTreeNode is the constructor of TreeNode
func CreateTreeNode(data []byte) *TreeNode {
	n := TreeNode{data: data, Parent: nil}
	n.Children = make([]*TreeNode, 0)
	n.TreeSize = 1
	n.LeafSize = 0
	n.Depth = -1
	return &n
}

// LoadData loads the node raw data from IPFS network lazily
func (n *TreeNode) Data() (data []byte, err error) {
	if len(n.data) == 0 && n.connector != nil && len(n.CID) > 0 {
		var myData []byte
		myData, err = n.connector.shell.BlockGet(n.CID)
		if err != nil {
			return
		}
		n.data = myData
	}
	data = n.data

	return data, err
}

// AddChild links a child to the current node
func (n *TreeNode) AddChild(child *TreeNode) {
	n.Children = append(n.Children, child)
	n.TreeSize += child.TreeSize
	n.LeafSize += child.LeafSize
	child.Parent = n
	child.Depth = n.Depth + 1
}

// GetFlattenedTree removes dependencies inside lattice windows and returns an array of tree nodes
func (n *TreeNode) GetFlattenedTree(s int, p int, swap bool) []*TreeNode {
	nodes := make([]*TreeNode, n.TreeSize)

	// preorder traversal of the tree
	internals := make([]*TreeNode, 0)
	var walker func(*TreeNode)
	walker = func(parent *TreeNode) {
		if parent == nil {
			return
		}
		nodes[parent.PreOrderIdx] = parent
		if len(parent.Children) > 0 {
			for _, child := range parent.Children {
				walker(child)
			}
			// meaningless to include root
			if parent != n {
				internals = append(internals, parent)
			}
		}
	}
	walker(n)

	if swap {
		nodes = n.swapFlattenedTree(nodes, internals, s, p)
	}
	return nodes
}

func (n *TreeNode) swapFlattenedTree(nodes []*TreeNode, internals []*TreeNode, s int, p int) []*TreeNode {
	// move the parents at least one LW away from their children
	windowSize := s * p
	for _, internalNode := range internals {
		lowestChild := internalNode.Children[0]
		highestChild := internalNode.Children[len(internalNode.Children)-1]
		for j := windowSize; j < n.TreeSize; j += s {
			inWindow := (nodes[j].PreOrderIdx > lowestChild.PreOrderIdx-windowSize &&
				nodes[j].PreOrderIdx < highestChild.PreOrderIdx+windowSize)
			if !inWindow && len(nodes[j].Children) == 0 {
				// Swap position of internalNode and the data
				nodes[j], nodes[internalNode.PreOrderIdx-1] = nodes[internalNode.PreOrderIdx-1], nodes[j]
				break
			}
		}
	}

	return nodes
}

func (n *EmptyTreeNode) swapUnflattenedTree(s int, p int, size int) {
	// Collect all internal nodes for swapping
	var internals []*EmptyTreeNode
	var walker func(node *EmptyTreeNode)
	walker = func(node *EmptyTreeNode) {
		if node == nil {
			return
		}
		if len(node.Children) > 0 {
			// Add to internals if not root
			if node != n {
				internals = append(internals, node)
			}
			for _, child := range node.Children {
				walker(child)
			}
		}
	}
	walker(n)

	// Replicate the current nodes array based on PreOrderIdx index
	nodes := make([]*EmptyTreeNode, size)
	var collectNodes func(node *EmptyTreeNode)
	collectNodes = func(node *EmptyTreeNode) {
		if node == nil {
			return
		}
		nodes[node.PreOrderIdx] = node
		for _, child := range node.Children {
			collectNodes(child)
		}
	}
	collectNodes(n)

	windowSize := s * p
	for _, internalNode := range internals {
		lowestChild := internalNode.Children[0]
		highestChild := internalNode.Children[len(internalNode.Children)-1]
		for j := windowSize; j < size; j += s {
			inWindow := (nodes[j].PreOrderIdx > lowestChild.PreOrderIdx-windowSize &&
				nodes[j].PreOrderIdx < highestChild.PreOrderIdx+windowSize)
			if !inWindow && len(nodes[j].Children) == 0 {
				// Swap the PreOrderIdx of internalNode and the data
				nodes[j].LatticeIdx, nodes[internalNode.PreOrderIdx-1].LatticeIdx = nodes[internalNode.PreOrderIdx-1].LatticeIdx, nodes[j].LatticeIdx
				break
			}
		}
	}
}

// GetLeafNodes returns all tree leaves
func (n *TreeNode) GetLeafNodes() []*TreeNode {
	nodes := make([]*TreeNode, n.LeafSize)

	// preorder traversal of the tree
	var walker func(*TreeNode)
	var leafCnt = 0
	walker = func(parent *TreeNode) {
		if parent == nil {
			return
		}
		if len(parent.Children) > 0 {
			for _, child := range parent.Children {
				walker(child)
			}
		} else {
			nodes[leafCnt] = parent
			leafCnt++
		}
	}
	walker(n)
	return nodes
}

// TODO:VERY IMPORTANT - update the tree construction logic
// now we only need number of leaves + max children
// 1. divide the total leaves over max children + 1 (if remainder is not zero) => that's the first level
// 2. for each level, if the number of nodes is greater than max children, then create another level
// 3. we keep doing that until the number of nodes we've created in this level are less than max children, that's root
// 4. need to keep track of the depth of the tree, verify with the stored depth
// 5. it would also be helpful to verify total number of nodes is same as the stored number of nodes

// Construct an empty tree given the following
// 1. the number of leaves: L
// 2. Maximum depth: D
// 3. Maximum number of children per node: K
// 4. Total Number of nodes: N
// Returns a tree with N nodes
// ConstructTree constructs the tree as described
func ConstructTree(L, K, D, N, s, p int) (*EmptyTreeNode, map[int]int, map[int]*EmptyTreeNode, error) {

	// create a map from lattice index of child to parent
	child_parent := make(map[int]int)

	// map lattice index to tree node
	index_map := make(map[int]*EmptyTreeNode)

	var map_nodes func(*EmptyTreeNode)
	map_nodes = func(node *EmptyTreeNode) {
		if node == nil {
			return
		}

		index_map[node.LatticeIdx] = node
		for _, child := range node.Children {
			child_parent[child.LatticeIdx] = node.LatticeIdx
			map_nodes(child)
		}
	}

	currentPreOrderIdx := 0
	var assignPreOrderIndex func(*EmptyTreeNode)
	// Assign pre-order index to nodes
	assignPreOrderIndex = func(node *EmptyTreeNode) {
		if node == nil {
			return
		}
		node.PreOrderIdx = currentPreOrderIdx
		node.LatticeIdx = currentPreOrderIdx
		currentPreOrderIdx++
		for _, child := range node.Children {
			assignPreOrderIndex(child)
		}
	}

	// Create initial set of leaves
	nodes := make([]*EmptyTreeNode, L)
	for i := 0; i < L; i++ {
		nodes[i] = &EmptyTreeNode{}
	}

	totalNodes := L
	currentDepth := 1

	// Build tree bottom-up until only one node (root) remains
	for len(nodes) > 1 && currentDepth <= D {
		var parents []*EmptyTreeNode
		for i := 0; i < len(nodes); i += K {
			end := i + K
			if end > len(nodes) {
				end = len(nodes)
			}

			parent := &EmptyTreeNode{}
			for j := i; j < end; j++ {
				nodes[j].Parent = parent
				parent.Children = append(parent.Children, nodes[j])
			}
			parents = append(parents, parent)
			totalNodes++
		}
		nodes = parents
		currentDepth++
	}

	if len(nodes) != 1 {
		return nil, nil, nil, errors.New("unexpected error in tree construction")
	}

	// Check if the constructed tree meets the expected depth and total nodes
	if currentDepth != D || totalNodes != N {
		return nil, nil, nil, fmt.Errorf("constructed tree does not meet the criteria. Depth: %d, Nodes: %d", currentDepth, totalNodes)
	}

	root := nodes[0]

	// Assign pre-order indexes top-down
	assignPreOrderIndex(root)

	// Swap the lattice idices
	root.swapUnflattenedTree(s, p, N)

	child_parent[root.LatticeIdx] = root.LatticeIdx
	map_nodes(root)

	return root, child_parent, index_map, nil
}

func CreateParityTree(L, K int) (*ParityTreeNode, map[int]*ParityTreeNode) {
	// Create L leaves
	leaves := make([]*ParityTreeNode, L)
	for i := range leaves {
		leaves[i] = &ParityTreeNode{Index: i}
	}

	// Map for leaf indices to nodes
	leafMap := make(map[int]*ParityTreeNode)
	for _, leaf := range leaves {
		leafMap[leaf.Index] = leaf
	}

	// Initialize the current level with leaves
	currentLevel := leaves
	nextParentIndex := L

	for len(currentLevel) > 1 {
		var newLevel []*ParityTreeNode
		var tempNodes []*ParityTreeNode

		// Group nodes to assign them to new parents
		for _, node := range currentLevel {
			tempNodes = append(tempNodes, node)
			if len(tempNodes) == K || (node == currentLevel[len(currentLevel)-1] && len(tempNodes) > 0) {
				parent := &ParityTreeNode{Index: nextParentIndex}
				nextParentIndex++
				parent.Children = tempNodes
				for _, child := range tempNodes {
					child.Parent = parent
				}
				newLevel = append(newLevel, parent)
				tempNodes = []*ParityTreeNode{}
			}
		}
		currentLevel = newLevel
	}

	// The root of the tree is the last node created
	root := currentLevel[0]

	return root, leafMap
}
