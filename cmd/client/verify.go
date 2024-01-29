package client

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zale144/fileserver/internal/client"
)

// VerifyCmd represents the verify command
var VerifyCmd = &cobra.Command{
	Use:   "verify [filePath] [proofPath] [root]",
	Short: "Verify the integrity of a downloaded file",
	Long: `Verify the file integrity using a Merkle proof.
For example:

fileserver verify downloaded_file.txt proof.json`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		proofPath := args[1]
		root := args[2]

		valid, err := client.VerifyFile(filePath, proofPath, root)
		if err != nil {
			return fmt.Errorf("failed to verify file: %w", err)
		}

		if valid {
			fmt.Println("The file is valid.")
		} else {
			fmt.Println("The file is NOT valid.")
		}

		return nil
	},
}
