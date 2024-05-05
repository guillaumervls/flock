package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

type flockRootFlags struct {
	flyTomlGlobs []string
	envFiles     []string
	org          string
}

var rootFlags flockRootFlags

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "flock",
	Short: "Deploy multiple Fly.io applications",
	Long: `Deploy multiple Fly.io applications (defined in their respectives "fly.toml"s) together.
You need to have the Fly.io CLI installed and authenticated to use this tool.
	`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	log.SetFlags(0) // Remove timestamp from log messages

	rootCmd.PersistentFlags().StringSliceVar(
		&rootFlags.flyTomlGlobs,
		"flytoml-glob",
		[]string{"**/fly.toml", "**/*.fly.toml"},
		"Glob patterns to find fly.toml files (default: **/fly.toml,**/*.fly.toml)",
	)

	rootCmd.PersistentFlags().StringSliceVar(
		&rootFlags.envFiles,
		"env-file",
		[]string{".env"},
		"File(s) to read and save (in the last one) environment variables (default: .env)",
	)

	rootCmd.PersistentFlags().StringVar(
		&rootFlags.org,
		"org",
		"personal",
		"Fly.io organization name to operate in (default: personal)",
	)

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
