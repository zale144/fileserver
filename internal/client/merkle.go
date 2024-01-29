package client

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zale144/fileserver/internal/merkle"
)

func MerkleRoot(directory string, dataSize int) (string, error) {
	hashChan := make(chan []byte)
	defer close(hashChan)

	go func() {
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				hasher := sha256.New()
				if _, err := io.Copy(hasher, file); err != nil {
					return err
				}
				hash := hasher.Sum(nil)
				hashChan <- hash
			}
			return nil
		})
		if err != nil {
			fmt.Printf("failed to walk directory: %v\n", err)
			return
		}
	}()

	tree := merkle.NewTreeFromStream(hashChan, dataSize)
	rootHash := tree.RootHash()

	file, err := os.Create("merkle_root")
	if err != nil {
		return "", fmt.Errorf("failed to create merkle_root file: %w", err)
	}
	defer file.Close()
	if _, err := file.WriteString(rootHash); err != nil {
		return "", fmt.Errorf("failed to write Merkle root to file: %w", err)
	}
	return rootHash, nil
}
