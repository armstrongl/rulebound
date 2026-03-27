package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/larah/rulebound/internal/config"
	hugobuilder "github.com/larah/rulebound/internal/hugo"
	"github.com/larah/rulebound/internal/parser"
	"github.com/spf13/cobra"
)

// Build flags.
var (
	buildOutput string
	buildConfig string
	buildHugo   string
	buildStrict bool
)

// buildCmd defines the `rulebound build` sub-command.
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

// runBuild executes the `rulebound build` pipeline:
// validate → load config → parse rules → scaffold → Hugo build → Pagefind → done.
func runBuild(cmd *cobra.Command, args []string) error {
	packagePath := args[0]

	// ── Validate package path ─────────────────────────────────────────────
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

	// ── Load config ───────────────────────────────────────────────────────
	var cfg *config.Config
	if buildConfig != "" {
		cfg, err = config.LoadFile(buildConfig)
	} else {
		cfg, err = config.Load(packagePath)
	}
	if err != nil {
		return &exitError{code: ExitConfigError, err: fmt.Errorf("loading config: %w", err)}
	}

	if Verbose {
		fmt.Printf("Package:  %s\n", packagePath)
		fmt.Printf("Output:   %s\n", buildOutput)
		fmt.Printf("Title:    %s\n", cfg.Title)
		fmt.Printf("BaseURL:  %s\n", cfg.BaseURL)
		fmt.Printf("Strict:   %v\n", buildStrict)
		fmt.Printf("Hugo:     %s\n", buildHugo)
	}

	// ── Parse rules ───────────────────────────────────────────────────────
	result, err := parser.ParsePackage(packagePath)
	if err != nil {
		return fmt.Errorf("parsing package: %w", err)
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s: %s\n", w.File, w.Message)
	}

	if len(result.Rules) == 0 {
		return fmt.Errorf("no valid rules found in %s", packagePath)
	}

	// In strict mode, treat parse warnings as errors.
	if buildStrict && len(result.Warnings) > 0 {
		return &exitError{
			code: ExitGeneral,
			err:  fmt.Errorf("strict mode: %d parse warning(s) found", len(result.Warnings)),
		}
	}

	// ── Find and verify Hugo ──────────────────────────────────────────────
	hugoBin, err := hugobuilder.FindHugo(buildHugo)
	if err != nil {
		return mapBuildError(err)
	}

	hugoVer, err := hugobuilder.CheckHugoVersion(hugoBin)
	if err != nil {
		return mapBuildError(err)
	}

	if Verbose {
		fmt.Printf("Hugo:     %s (version %s)\n", hugoBin, hugoVer)
	}

	// ── Scaffold Hugo project ─────────────────────────────────────────────
	scaffold, err := hugobuilder.Scaffold(result, cfg, packagePath)
	if err != nil {
		if scaffold != nil && scaffold.TempDir != "" {
			os.RemoveAll(scaffold.TempDir)
		}
		return fmt.Errorf("scaffolding Hugo project: %w", err)
	}

	// Register a signal handler before the build so the temp dir is cleaned
	// up if the user presses Ctrl-C during Hugo execution.
	// defer os.RemoveAll does not run on os.Exit or log.Fatal.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\nInterrupted, cleaning up...")
			os.RemoveAll(scaffold.TempDir)
			os.Exit(1)
		case <-ctx.Done():
			// Normal exit — cleanup handled by defer below.
		}
	}()

	defer os.RemoveAll(scaffold.TempDir)

	if Verbose {
		fmt.Printf("Temp dir: %s\n", scaffold.TempDir)
	}

	// ── Resolve output directory ──────────────────────────────────────────
	outputDir, err := filepath.Abs(buildOutput)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	// ── Hugo build ────────────────────────────────────────────────────────
	buildResult, err := hugobuilder.Build(hugoBin, scaffold.TempDir, outputDir)
	if err != nil {
		if Verbose && buildResult != nil {
			if buildResult.Stdout != "" {
				fmt.Printf("Hugo stdout:\n%s\n", buildResult.Stdout)
			}
			if buildResult.Stderr != "" {
				fmt.Fprintf(os.Stderr, "Hugo stderr:\n%s\n", buildResult.Stderr)
			}
		}
		return mapBuildError(err)
	}

	if Verbose && buildResult.Stderr != "" {
		fmt.Fprintf(os.Stderr, "Hugo output:\n%s", buildResult.Stderr)
	}

	// ── Pagefind ──────────────────────────────────────────────────────────
	found, err := hugobuilder.RunPagefind(outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: pagefind indexing failed: %v\n", err)
	} else if !found {
		if Verbose {
			fmt.Fprintln(os.Stderr, "Note: pagefind not found on $PATH; search index not generated")
		}
	} else if Verbose {
		fmt.Println("Pagefind search index generated")
	}

	// ── Summary ───────────────────────────────────────────────────────────
	total := len(result.Rules) + len(result.Warnings)
	skipped := len(result.Warnings)
	fmt.Printf("Build complete: %d/%d rules processed, %d skipped", len(result.Rules), total, skipped)
	if skipped > 0 {
		fmt.Print(" (see warnings above)")
	}
	fmt.Println(".")
	fmt.Printf("Output: %s\n", outputDir)

	return nil
}

// exitError pairs an error with a specific exit code for the CLI.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	return e.err.Error()
}

func (e *exitError) Unwrap() error {
	return e.err
}

// mapBuildError converts a *hugo.BuildError to an *exitError for the CLI layer.
// Other error types pass through unchanged.
func mapBuildError(err error) error {
	if be, ok := err.(*hugobuilder.BuildError); ok {
		return &exitError{code: be.ExitCode, err: be}
	}
	return err
}
