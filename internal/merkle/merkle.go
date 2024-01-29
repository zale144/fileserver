package merkle

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync"
)

type Tree struct {
	Root       *node
	Proofs     [][][]byte
	Depth      int
	leafs      []*node
	numWorkers int
}

type Proof struct {
	Proof [][]byte
}

type node struct {
	Hash   []byte
	Left   *node
	Right  *node
	Parent *node
	Index  int // This field keeps track of the index of the data block the node corresponds to
}

type nodeResult struct {
	node *node
}

type nodeResultBatch struct {
	results []*nodeResult
}

type proofResultBatch struct {
	results []proofResult
}

type proofResult struct {
	proof [][]byte
	index int
}

var defaultNumWorkers = runtime.NumCPU() * 8

func NewTree(dataBlocks [][]byte) *Tree {
	dataBlocks = padDataBlocks(dataBlocks)
	depth := 0
	lenData := len(dataBlocks)

	for size := lenData; size > 1; size = size / 2 {
		depth++
	}

	numWorkers := defaultNumWorkers
	if lenData < numWorkers {
		numWorkers = lenData // Adjust the number of workers
	}
	numWorkers = nextPowerOfTwo(numWorkers) // Make sure the number of workers is a power of 2

	t := &Tree{
		Proofs:     make([][][]byte, lenData),
		Depth:      depth,
		leafs:      make([]*node, lenData),
		numWorkers: numWorkers,
	}

	t.buildTree(dataBlocks)
	t.generateProofs()
	return t
}

func NewTreeFromStream(dataBlocks <-chan []byte, lenData int) *Tree {
	depth := 0

	for size := lenData; size > 1; size = size / 2 {
		depth++
	}

	numWorkers := defaultNumWorkers
	if lenData < numWorkers {
		numWorkers = lenData // Adjust the number of workers
	}
	numWorkers = nextPowerOfTwo(numWorkers) // Make sure the number of workers is a power of 2

	t := &Tree{
		Proofs:     make([][][]byte, lenData),
		Depth:      depth,
		leafs:      make([]*node, lenData),
		numWorkers: numWorkers,
	}

	t.buildTreeFromStream(dataBlocks, lenData)
	t.padLeafs()
	t.generateProofs()
	return t
}

func (t *Tree) buildTree(dataBlocks [][]byte) {
	if len(dataBlocks) == 0 {
		return
	}

	t.buildLeaves(dataBlocks)
	t.buildBranches()
}

func (t *Tree) buildTreeFromStream(dataBlocks <-chan []byte, dataSize int) {
	t.buildLeavesFromStream(dataBlocks, dataSize)
	t.buildBranches()
}

func (t *Tree) buildLeaves(dataBlocks [][]byte) {
	// Hash the data blocks concurrently using worker pool
	numWorkers := t.numWorkers
	hashResults := make(chan *nodeResultBatch, numWorkers)
	batchSize := len(dataBlocks) / numWorkers

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		start := i * batchSize
		end := start + batchSize
		go leafWorker(newLeafNode, dataBlocks[start:end], start, hashResults, &wg)
	}

	go func() {
		wg.Wait()
		close(hashResults)
	}()
	for res := range hashResults {
		for _, nRes := range res.results {
			t.leafs[nRes.node.Index] = nRes.node
		}
	}
}

func (t *Tree) buildLeavesFromStream(dataBlocks <-chan []byte, dataSize int) {
	// Hash the data blocks concurrently using worker pool
	numWorkers := t.numWorkers
	hashResults := make(chan *nodeResultBatch, numWorkers)
	batchSize := dataSize / numWorkers

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		start := i * batchSize
		end := start + batchSize
		batch := make([][]byte, batchSize)
		for j := start; j < end; j++ {
			batch[j-start] = <-dataBlocks
		}
		go leafWorker(newLeafNodeFromHash, batch, start, hashResults, &wg)
	}

	go func() {
		wg.Wait()
		close(hashResults)
	}()
	for res := range hashResults {
		for _, nRes := range res.results {
			t.leafs[nRes.node.Index] = nRes.node
		}
	}
}

// Worker pool for hashing data blocks
func leafWorker(fn nodeFunc, dataBlocks [][]byte, startIndex int, results chan<- *nodeResultBatch, wg *sync.WaitGroup) {
	defer wg.Done()
	batch := &nodeResultBatch{results: make([]*nodeResult, len(dataBlocks))}
	for i, data := range dataBlocks {
		batch.results[i] = &nodeResult{
			node: fn(data, startIndex+i),
		}
	}
	results <- batch
}

type nodeFunc func(data []byte, index int) *node

func newLeafNode(data []byte, index int) *node {
	dataHashed := HashData(data)
	return &node{
		Hash:  dataHashed,
		Index: index,
	}
}

func newLeafNodeFromHash(dataHashed []byte, index int) *node {
	return &node{
		Hash:  dataHashed,
		Index: index,
	}
}

func (t *Tree) buildBranches() {
	var wg sync.WaitGroup
	nodes := t.leafs
	numWorkers := t.numWorkers
	// Concurrently build branches using worker pool
	for lenNodes := len(nodes); lenNodes > 1; {
		// Building current level will have half the number of nodes as the previous level
		branchResults := make(chan *nodeResultBatch, lenNodes/2)
		if lenNodes <= numWorkers {
			numWorkers = lenNodes / 2 // Adjust the number of workers
		}

		batchSize := lenNodes / numWorkers // batchSize has to be at least 2
		for i := 0; i < numWorkers; i++ {
			start := i * batchSize
			end := start + batchSize
			wg.Add(1)
			go t.branchWorker(nodes[start:end], start, branchResults, &wg)
			start = end
		}

		nextLevelNodes := make([]*node, lenNodes/2)
		go func() {
			wg.Wait()
			close(branchResults)
		}()
		for res := range branchResults {
			for _, nRes := range res.results {
				nextLevelNodes[nRes.node.Index] = nRes.node // Use the index from the result to ensure consistent ordering
			}
		}
		nodes, lenNodes = nextLevelNodes, len(nextLevelNodes)
	}
	t.Root = nodes[0]
}

// Worker pool for building branches
func (t *Tree) branchWorker(nodes []*node, startIndex int, results chan<- *nodeResultBatch, wg *sync.WaitGroup) {
	defer wg.Done()
	batch := &nodeResultBatch{results: make([]*nodeResult, len(nodes)/2)}
	for i := 0; i < len(nodes); i += 2 {
		left, right := nodes[i], nodes[i+1]
		branchNode := newBranchNode(left, right, (startIndex+i)/2)
		left.Parent, right.Parent = branchNode, branchNode
		batch.results[i/2] = &nodeResult{
			node: branchNode,
		}
	}
	results <- batch
}

func newBranchNode(left, right *node, index int) *node {
	dataHashed := merkleHash(left.Hash, right.Hash)
	return &node{
		Hash:  dataHashed,
		Left:  left,
		Right: right,
		Index: index,
	}
}

// Compute the Merkle hash of two child hashes
func merkleHash(left, right []byte) []byte {
	combinedData := make([]byte, 0, 64)
	combinedData = append(combinedData, left...)
	combinedData = append(combinedData, right...)
	return HashData(combinedData)
}

// HashData computes SHA256 hash
func HashData(data []byte) []byte {
	bytes := sha256.Sum256(data)
	return bytes[:]
}

func (t *Tree) generateProofs() {
	numWorkers := t.numWorkers
	proofChan := make(chan *proofResultBatch, numWorkers)
	batchSize := len(t.leafs) / numWorkers

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go t.proofWorker(batchSize, i, proofChan, &wg)
	}
	go func() {
		wg.Wait()
		close(proofChan)
	}()
	for proof := range proofChan {
		for _, p := range proof.results {
			t.Proofs[p.index] = p.proof
		}
	}
}

func (t *Tree) proofWorker(batchSize, idx int, results chan<- *proofResultBatch, wg *sync.WaitGroup) {
	defer wg.Done()
	start := idx * batchSize
	batch := &proofResultBatch{results: make([]proofResult, batchSize)}
	for i := 0; i < batchSize; i++ {
		index := start + i
		batch.results[i].proof = t.generateProof(index)
		batch.results[i].index = index
	}
	results <- batch
}

func (t *Tree) generateProof(idx int) [][]byte {
	proof := make([][]byte, t.Depth)

	leaf := t.leafs[idx]
	if leaf == nil {
		return nil
	}
	currentNode := leaf
	dataIndex := leaf.Index

	for i := 0; i < t.Depth; i++ {
		// Determine if our path is to the left or right based on dataIndex
		if dataIndex%2 == 0 { // Even index means left
			proof[i] = currentNode.Parent.Right.Hash
		} else { // Odd index means right
			proof[i] = currentNode.Parent.Left.Hash
		}
		// Move to the next level in the tree
		currentNode = currentNode.Parent
		dataIndex = currentNode.Index
	}

	return proof
}

func (t *Tree) RootHash() string {
	return fmt.Sprintf("%x", t.Root.Hash)
}

func VerifyProof(index int, hash []byte, proof [][]byte, rootHash []byte) bool {
	for _, step := range proof {
		if index%2 == 0 {
			hash = merkleHash(hash, step)
			index = index / 2
		} else {
			hash = merkleHash(step, hash)
			index = (index - 1) / 2
		}
	}
	return [32]byte(hash) == [32]byte(rootHash)
}

func (t *Tree) padLeafs() {
	count := len(t.leafs)
	next := nextPowerOfTwo(count)
	if count < next {
		paddedLeafs := make([]*node, next)
		copy(paddedLeafs, t.leafs)
		for ; count < next; count++ {
			paddedLeafs[count] = &node{}
		}
		t.leafs = paddedLeafs
	}
}

func padDataBlocks(dataBlocks [][]byte) [][]byte {
	count := len(dataBlocks)
	next := nextPowerOfTwo(count)
	if count < next {
		paddedData := make([][]byte, next)
		copy(paddedData, dataBlocks)
		for ; count < next; count++ {
			paddedData[count] = []byte{}
		}
		dataBlocks = paddedData
	}
	return dataBlocks
}

func nextPowerOfTwo(num int) int {
	next := 1
	for next < num {
		next <<= 1
	}
	return next
}
