// Package cmd implements the CLI commands for rulebound.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is the current rulebound version, replaced at build time via
// -ldflags "-X github.com/armstrongl/rulebound/cmd.Version=<ver>".
var Version = "dev"

// Verbose holds the global --verbose flag value. All sub-commands read it.
var Verbose bool

// Exit codes per plan spec.
const (
	ExitSuccess     = 0
	ExitGeneral     = 1
	ExitConfigError = 2
	ExitHugoError   = 3
	ExitHugoBuild   = 4
)

// rootCmd is the base command when rulebound is called with no sub-commands.
var rootCmd = &cobra.Command{
	Use:   "rulebound",
	Short: "Generate static style guide websites from Vale linting packages",
	Long: `rulebound reads a Vale rule package and generates a static website
documenting every rule, organised into categories, and powered by a Hugo theme.

Usage example:
  rulebound build ./my-vale-package --output ./public/`,
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits on failure.
// An *exitError carries a specific exit code; other errors exit with ExitGeneral.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		code := ExitGeneral
		if ee, ok := err.(*exitError); ok {
			code = ee.code
		}
		os.Exit(code)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Print verbose output")

	// Add sub-commands.
	rootCmd.AddCommand(buildCmd)
}
