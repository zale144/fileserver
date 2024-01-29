package client

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zale144/fileserver/internal/client"
)

// DownloadCmd represents the download command
var DownloadCmd = &cobra.Command{
	Use:   "download [fileID] [url]", // TODO: get url from config
	Short: "Download a file from the server",
	Long: `Download requests a file and its Merkle proof from the server.
For example:

fileserver download file123`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileID := args[0]
		url := args[1]
		err := client.DownloadFile(fileID, url)
		if err != nil {
			return fmt.Errorf("failed to download file: %w", err)
		}
		fmt.Printf("Successfully downloaded file with ID %s\n", fileID)
		return nil
	},
}
