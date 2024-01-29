package client

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zale144/fileserver/internal/client"
)

// UploadCmd represents the upload command
var UploadCmd = &cobra.Command{
	Use:   "upload [dir] [url]",
	Short: "Upload multiple files to the server",
	Long: `Upload allows the client to send multiple files to the server.
For example:

fileserver upload ./directory http://localhost:8080/file`, // TODO: get url from config
	RunE: func(cmd *cobra.Command, args []string) error {
		dirPath := args[0]
		url := args[1]
		if err := client.UploadDirectory(dirPath, url); err != nil {
			return fmt.Errorf("failed to upload %s: %w", dirPath, err)
		}
		fmt.Printf("Successfully uploaded %s\n", dirPath)
		return nil
	},
}
