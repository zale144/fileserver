package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zale144/fileserver/cmd/client"
	"github.com/zale144/fileserver/cmd/fileserver"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "fileserver",
	Short: "fileserver is a CLI application for file management with Merkle tree verification",
	Long:  `Fast and Flexible.`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.AddCommand(fileserver.ServerCmd)
	RootCmd.AddCommand(client.UploadCmd)
	RootCmd.AddCommand(client.DownloadCmd)
	RootCmd.AddCommand(client.VerifyCmd)
	RootCmd.AddCommand(client.MerkleRootCmd)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.client.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigName(".client")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
