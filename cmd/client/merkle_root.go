package client

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zale144/fileserver/internal/client"
)

// MerkleRootCmd represents the merkle command
var MerkleRootCmd = &cobra.Command{
	Use:   "merkle [directory]",
	Short: "Create a Merkle root from files in a directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		directory := args[0]
		dataSize, err := countFilesInDir(directory)
		if err != nil {
			return fmt.Errorf("failed to count files in directory: %w", err)
		}
		rootHash, err := client.MerkleRoot(directory, dataSize)
		if err != nil {
			return fmt.Errorf("failed to create Merkle tree: %w", err)
		}

		fmt.Printf("Merkle Root: %s\n", rootHash)
		return nil
	},
}

func countFilesInDir(directory string) (int, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0, err
	}

	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			fileCount++
		}
	}

	return fileCount, nil
}
