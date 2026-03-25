package cmd

import (
	"fmt"
	"os"

	"github.com/larah/rulebound/internal/config"
	"github.com/spf13/cobra"
)

// Build flags.
var (
	buildOutput string
	buildConfig string
	buildHugo   string
	buildStrict bool
)

// buildCmd is the `rulebound build` sub-command.
var buildCmd = &cobra.Command{
	Use:   "build <package-path>",
	Short: "Build a static style guide website from a Vale rule package",
	Long: `build reads the Vale rule package at <package-path>, parses every rule,
generates Hugo content files, and runs Hugo to produce a static website.

The package path must be a directory containing Vale YAML rule files.`,
	Args: cobra.ExactArgs(1),
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "./public/", "Output directory for the generated site")
	buildCmd.Flags().StringVarP(&buildConfig, "config", "c", "", "Path to rulebound.yml (default: auto-detect in package root)")
	buildCmd.Flags().StringVar(&buildHugo, "hugo", "", "Path to Hugo binary (default: auto-detect on $PATH)")
	buildCmd.Flags().BoolVar(&buildStrict, "strict", false, "Treat warnings as errors")
}

// runBuild is the entry point for `rulebound build`.
// Phase 1: validate inputs and load config. The actual build pipeline is wired in Phase 5.
func runBuild(cmd *cobra.Command, args []string) error {
	packagePath := args[0]

	// Validate that the package path exists and is a directory.
	info, err := os.Stat(packagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("package path does not exist: %s", packagePath)
		}
		return fmt.Errorf("checking package path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("package path must be a directory, got: %s", packagePath)
	}

	// Determine config directory: either the explicit --config flag's parent, or the package root.
	configDir := packagePath
	if buildConfig != "" {
		configDir = buildConfig
	}

	// Load (or default) the rulebound.yml configuration.
	cfg, err := config.Load(configDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if Verbose {
		fmt.Printf("Package:  %s\n", packagePath)
		fmt.Printf("Output:   %s\n", buildOutput)
		fmt.Printf("Title:    %s\n", cfg.Title)
		fmt.Printf("BaseURL:  %s\n", cfg.BaseURL)
		fmt.Printf("Strict:   %v\n", buildStrict)
		fmt.Printf("Hugo:     %s\n", buildHugo)
	}

	// Phase 5 will wire the full build pipeline here.
	fmt.Println("rulebound build: not yet implemented")
	return nil
}
