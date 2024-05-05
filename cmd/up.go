package cmd

import (
	"log"

	"github.com/guillaumervls/flock/pkg/flock"
	"github.com/spf13/cobra"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Deploy multiple Fly.io applications",
	Long:  `Deploy multiple Fly.io applications (defined in their respectives "fly.toml"s) together.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := flock.Up(rootFlags.flyTomlGlobs, rootFlags.envFiles, rootFlags.org)
		if err != nil {
			log.Fatalf("Error deploying: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// upCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// upCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
