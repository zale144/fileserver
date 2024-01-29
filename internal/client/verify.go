package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/zale144/fileserver/internal/merkle"
)

func VerifyFile(filePath, proofPath, rootPath string) (bool, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	proof, index, err := getProof(proofPath)
	if err != nil {
		return false, fmt.Errorf("failed to get proof: %w", err)
	}

	root, err := getMerkleRoot(rootPath)
	if err != nil {
		return false, fmt.Errorf("failed to get root: %w", err)
	}

	valid := merkle.VerifyProof(index, merkle.HashData(fileContent), proof, root)
	if !valid {
		return false, fmt.Errorf("file verification failed")
	}
	return valid, nil
}

func getProof(proofPath string) ([][]byte, int, error) {
	proofContent, err := os.ReadFile(proofPath)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to read proof file: %w", err)
	}

	proof := new(MerkleProof)
	if err = json.Unmarshal(proofContent, proof); err != nil {
		return nil, -1, fmt.Errorf("failed to unmarshal proof: %w", err)
	}

	proofBytes := make([][]byte, len(proof.Proof))
	for i, p := range proof.Proof {
		decodedBytes := make([]byte, hex.DecodedLen(len(p)))
		_, err := hex.Decode(decodedBytes, []byte(p))
		if err != nil {
			return nil, -1, fmt.Errorf("failed to decode proof: %w", err)
		}
		proofBytes[i] = decodedBytes
	}

	return proofBytes, int(proof.Index), nil
}

func getMerkleRoot(rootPath string) ([]byte, error) {
	root, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read proof file: %w", err)
	}

	decodedBytes := make([]byte, hex.DecodedLen(len(root)))
	_, err = hex.Decode(decodedBytes, root)
	if err != nil {
		return nil, fmt.Errorf("failed to decode root: %w", err)
	}
	return decodedBytes, nil
}
