package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "1.0.0"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("v" + version)
	},
}
