package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	hugobuilder "github.com/armstrongl/rulebound/internal/hugo"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// exitError type
// ---------------------------------------------------------------------------

func TestExitError_Error(t *testing.T) {
	inner := errors.New("something broke")
	ee := &exitError{code: ExitGeneral, err: inner}

	if got := ee.Error(); got != "something broke" {
		t.Fatalf("Error() = %q, want %q", got, "something broke")
	}
}

func TestExitError_Unwrap(t *testing.T) {
	inner := errors.New("wrapped")
	ee := &exitError{code: ExitHugoError, err: inner}

	if got := ee.Unwrap(); got != inner {
		t.Fatalf("Unwrap() = %v, want %v", got, inner)
	}
}

// ---------------------------------------------------------------------------
// mapBuildError
// ---------------------------------------------------------------------------

func TestMapBuildError_BuildError(t *testing.T) {
	be := &hugobuilder.BuildError{
		ExitCode: ExitHugoBuild,
		Message:  "hugo build failed",
		Err:      errors.New("exit status 1"),
	}

	got := mapBuildError(be)

	ee, ok := got.(*exitError)
	if !ok {
		t.Fatalf("mapBuildError returned %T, want *exitError", got)
	}
	if ee.code != ExitHugoBuild {
		t.Fatalf("exit code = %d, want %d", ee.code, ExitHugoBuild)
	}
	if ee.err != be {
		t.Fatalf("wrapped error = %v, want original BuildError %v", ee.err, be)
	}
}

func TestMapBuildError_RegularError(t *testing.T) {
	plain := errors.New("ordinary error")

	got := mapBuildError(plain)

	if got != plain {
		t.Fatalf("mapBuildError returned %v, want original error %v", got, plain)
	}
}

func TestMapBuildError_BuildErrorExitCode(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
	}{
		{"exit code 3 (hugo error)", ExitHugoError},
		{"exit code 4 (hugo build)", ExitHugoBuild},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be := &hugobuilder.BuildError{
				ExitCode: tt.exitCode,
				Message:  "test",
			}

			got := mapBuildError(be)

			ee, ok := got.(*exitError)
			if !ok {
				t.Fatalf("mapBuildError returned %T, want *exitError", got)
			}
			if ee.code != tt.exitCode {
				t.Fatalf("exit code = %d, want %d", ee.code, tt.exitCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Exit code constants
// ---------------------------------------------------------------------------

func TestExitCodeConstants(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"ExitSuccess", ExitSuccess, 0},
		{"ExitGeneral", ExitGeneral, 1},
		{"ExitConfigError", ExitConfigError, 2},
		{"ExitHugoError", ExitHugoError, 3},
		{"ExitHugoBuild", ExitHugoBuild, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// runBuild error paths (via cobra command execution)
// ---------------------------------------------------------------------------

// newBuildCommand creates a fresh copy of the build command for testing,
// avoiding shared state between tests.
func newBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build <package-path>",
		Short: "Build a static style guide website from a Vale rule package",
		Args:  cobra.ExactArgs(1),
		RunE:  runBuild,
	}
	cmd.Flags().StringVarP(&buildOutput, "output", "o", "./public/", "Output directory for the generated site")
	cmd.Flags().StringVarP(&buildConfig, "config", "c", "", "Path to rulebound.yml")
	cmd.Flags().StringVar(&buildHugo, "hugo", "", "Path to Hugo binary")
	cmd.Flags().BoolVar(&buildStrict, "strict", false, "Treat warnings as errors")
	return cmd
}

func TestRunBuild_MissingArgument(t *testing.T) {
	cmd := newBuildCommand()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no argument is provided, got nil")
	}
}

func TestRunBuild_NonExistentPath(t *testing.T) {
	cmd := newBuildCommand()
	cmd.SetArgs([]string{"/tmp/rulebound-nonexistent-path-xyz-12345"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}

	want := "does not exist"
	if got := err.Error(); !containsSubstring(got, want) {
		t.Fatalf("error = %q, want it to contain %q", got, want)
	}
}

func TestRunBuild_PathIsFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "notadir.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("setup: create temp file: %v", err)
	}

	cmd := newBuildCommand()
	cmd.SetArgs([]string{tmpFile})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when path is a file, got nil")
	}

	want := "must be a directory"
	if got := err.Error(); !containsSubstring(got, want) {
		t.Fatalf("error = %q, want it to contain %q", got, want)
	}
}

// containsSubstring checks whether s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
