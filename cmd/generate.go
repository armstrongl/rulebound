package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/armstrongl/rulebound/internal/mdgen"
	"github.com/spf13/cobra"
)

// Generate flags.
var (
	generateOutput string
	generateForce  bool
)

// generateCmd defines the `rulebound generate` sub-command.
var generateCmd = &cobra.Command{
	Use:   "generate <file.md>",
	Short: "Generate a Vale YAML rule file from a structured Markdown file",
	Long: `generate reads a structured Markdown file with YAML frontmatter and vale-*
fenced code blocks, and emits a Vale-compatible YAML rule file.

The input .md file uses frontmatter for rule metadata and vale-swap,
vale-tokens, or vale-exceptions fenced blocks for rule data.

Supported rule types: substitution, existence, occurrence, capitalization.`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "Output path (default: input stem + .yml); use '-' for stdout")
	generateCmd.Flags().BoolVar(&generateForce, "force", false, "Overwrite output file if it already exists")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Validate input is a regular file.
	info, err := os.Stat(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", inputPath)
		}
		return fmt.Errorf("checking input file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("input path is a directory, expected a file: %s", inputPath)
	}

	// Read input.
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading input file: %w", err)
	}

	// Parse.
	src, warnings, err := mdgen.ParseMarkdown(data)
	if err != nil {
		return &exitError{code: ExitGeneral, err: fmt.Errorf("parsing %s: %w", inputPath, err)}
	}

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w.Message)
	}

	// Emit YAML.
	yamlBytes, emitWarnings, err := mdgen.EmitYAML(src)
	if err != nil {
		return &exitError{code: ExitGeneral, err: fmt.Errorf("generating YAML: %w", err)}
	}

	for _, w := range emitWarnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w.Message)
	}

	// Resolve output path.
	outputPath := generateOutput
	if outputPath == "" {
		// Default: input stem + .yml in the same directory.
		ext := filepath.Ext(inputPath)
		outputPath = strings.TrimSuffix(inputPath, ext) + ".yml"
	}

	// Write output.
	if outputPath == "-" {
		// stdout
		_, err = os.Stdout.Write(yamlBytes)
		if err != nil {
			return fmt.Errorf("writing to stdout: %w", err)
		}
		return nil
	}

	// Check if output file exists.
	if !generateForce {
		if _, err := os.Stat(outputPath); err == nil {
			return &exitError{
				code: ExitGeneral,
				err:  fmt.Errorf("output file already exists: %s (use --force to overwrite)", outputPath),
			}
		}
	}

	if err := os.WriteFile(outputPath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	if Verbose {
		fmt.Printf("Input:    %s\n", inputPath)
		fmt.Printf("Extends:  %s\n", src.Extends)
	}
	fmt.Printf("Generated: %s\n", outputPath)

	return nil
}
