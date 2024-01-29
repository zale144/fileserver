package merkle

import (
	"crypto/sha256"
)

type simpleMerkleTree struct {
	RootNode *merkleNode
}

type merkleNode struct {
	Left  *merkleNode
	Right *merkleNode
	Data  []byte
}

func newMerkleNode(left, right *merkleNode, data []byte) *merkleNode {
	node := merkleNode{}
	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right
	return &node
}

func NewMerkleTree(data [][]byte) *simpleMerkleTree {
	var nodes []merkleNode
	data = padDataBlocks(data)
	for _, dat := range data {
		node := newMerkleNode(nil, nil, dat)
		nodes = append(nodes, *node)
	}
	for len(nodes) > 1 {
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}

		var level []merkleNode
		for i := 0; i < len(nodes); i += 2 {
			node := newMerkleNode(&nodes[i], &nodes[i+1], nil)
			level = append(level, *node)
		}
		nodes = level
	}
	return &simpleMerkleTree{&nodes[0]}
}
