package cmd

import (
	"fmt"
	"os"

	"github.com/armstrongl/rulebound/internal/parser"
	"github.com/spf13/cobra"
)

// validateCmd defines the `rulebound validate` sub-command.
var validateCmd = &cobra.Command{
	Use:   "validate <file.yml> [file2.yml ...]",
	Short: "Validate Vale YAML rule files for structural errors",
	Long: `validate reads one or more Vale rule YAML files and reports structural
errors: missing required fields, invalid extends values, and rule-type-specific
field errors — without invoking Hugo.

Exit code 0 if all files are valid, 1 if any file has errors.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	var totalErrors int
	validCount := 0

	for _, path := range args {
		if Verbose {
			fmt.Printf("Validating: %s\n", path)
		}

		errs, err := parser.ValidateRule(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			totalErrors++
			continue
		}

		if len(errs) == 0 {
			validCount++
			continue
		}

		totalErrors += len(errs)
		for _, ve := range errs {
			fmt.Fprintf(os.Stderr, "%s: %s: %s\n", path, ve.Field, ve.Message)
		}
	}

	fileCount := len(args)
	invalidCount := fileCount - validCount
	fmt.Printf("Validated %d file(s): %d valid, %d with errors\n", fileCount, validCount, invalidCount)

	if totalErrors > 0 {
		return &exitError{code: ExitGeneral, err: fmt.Errorf("%d validation error(s) found", totalErrors)}
	}

	return nil
}
