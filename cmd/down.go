package cmd

import (
	"github.com/guillaumervls/flock/pkg/flock"
	"github.com/spf13/cobra"
)

// downCmd represents the down command
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Destroy multiple Fly.io applications",
	Long:  `Destroy multiple Fly.io applications deployed by "flock up".`,

	Run: func(cmd *cobra.Command, args []string) {
		flock.Down(rootFlags.flyTomlGlobs, rootFlags.envFiles)
	},
}

func init() {
	rootCmd.AddCommand(downCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// downCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// downCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
