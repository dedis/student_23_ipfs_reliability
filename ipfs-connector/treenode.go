package ipfsconnector

type EmptyTreeNode struct {
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

// Construct an empty tree given the following
// 1. the number of leaves: L
// 2. Maximum depth: D
// 3. Maximum number of children per node: K
// 4. Total Number of nodes: N
// Returns a tree with N nodes
// ConstructTree constructs the tree as described
func ConstructTree(L, K, D, N, s, p int) *EmptyTreeNode {

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

	// Helper function to create a new node with specific depth
	createNode := func(depth int) *EmptyTreeNode {
		return &EmptyTreeNode{
			Depth: depth,
		}
	}

	// Create leaves
	var leaves []*EmptyTreeNode
	for i := 0; i < L; i++ {
		leaves = append(leaves, createNode(D))
	}

	nodes := leaves

	// TODO: Make sure the logic is correct
	// Construct tree from bottom up
	for depth := D; depth > 1; depth-- {
		var newNodes []*EmptyTreeNode
		for len(nodes) > K {
			children := nodes[:K]
			nodes = nodes[K:]

			node := createNode(depth - 1)
			node.Children = children
			for _, child := range children {
				child.Parent = node
			}
			newNodes = append(newNodes, node)
		}
		newNodes = append(newNodes, nodes...)
		nodes = newNodes
	}

	root := createNode(1)
	root.Children = nodes
	for _, node := range nodes {
		node.Parent = root
	}

	// Assign pre-order indexes top-down
	assignPreOrderIndex(root)

	// Swap the lattice idices
	root.swapUnflattenedTree(s, p, N)

	return root
}
