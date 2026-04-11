package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetGenerateFlags resets package-level flag variables to their defaults.
// Cobra flag values persist between rootCmd.Execute() calls in the same process.
func resetGenerateFlags(t *testing.T) {
	t.Helper()
	generateOutput = ""
	generateForce = false
}

// writeTestMD writes a minimal substitution .md to the given directory.
func writeTestMD(t *testing.T, dir, name string) string {
	t.Helper()
	content := `---
extends: substitution
message: "Prefer '%s' over '%s'."
level: warning
---

# Test

` + "```vale-swap\nleverage: use\n```\n"

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test md: %v", err)
	}
	return path
}

func TestGenerate_DefaultOutput(t *testing.T) {
	resetGenerateFlags(t)
	dir := t.TempDir()
	mdPath := writeTestMD(t, dir, "Jargon.md")

	rootCmd.SetArgs([]string{"generate", mdPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ymlPath := filepath.Join(dir, "Jargon.yml")
	data, err := os.ReadFile(ymlPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if !strings.Contains(string(data), "extends: substitution") {
		t.Errorf("output should contain extends: substitution")
	}
	if !strings.Contains(string(data), "leverage: use") {
		t.Errorf("output should contain swap entry")
	}
}

func TestGenerate_ExplicitOutput(t *testing.T) {
	resetGenerateFlags(t)
	dir := t.TempDir()
	mdPath := writeTestMD(t, dir, "Test.md")
	outPath := filepath.Join(dir, "custom.yml")

	rootCmd.SetArgs([]string{"generate", mdPath, "--output", outPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("output file not created at %s", outPath)
	}
}

func TestGenerate_Stdout(t *testing.T) {
	resetGenerateFlags(t)
	dir := t.TempDir()
	mdPath := writeTestMD(t, dir, "Test.md")

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"generate", mdPath, "--output", "-"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "extends: substitution") {
		t.Errorf("stdout should contain YAML output, got: %s", output)
	}
}

func TestGenerate_ForceOverwrite(t *testing.T) {
	resetGenerateFlags(t)
	dir := t.TempDir()
	mdPath := writeTestMD(t, dir, "Test.md")
	ymlPath := filepath.Join(dir, "Test.yml")

	// Create existing file.
	if err := os.WriteFile(ymlPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Without --force should fail.
	rootCmd.SetArgs([]string{"generate", mdPath})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when output exists without --force")
	}

	// With --force should succeed.
	rootCmd.SetArgs([]string{"generate", mdPath, "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}

	data, _ := os.ReadFile(ymlPath)
	if strings.Contains(string(data), "old content") {
		t.Error("file should have been overwritten")
	}
}

func TestGenerate_FileNotExist(t *testing.T) {
	resetGenerateFlags(t)
	rootCmd.SetArgs([]string{"generate", "/nonexistent/file.md"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestGenerate_InputIsDirectory(t *testing.T) {
	resetGenerateFlags(t)
	dir := t.TempDir()

	rootCmd.SetArgs([]string{"generate", dir})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when input is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention directory: %v", err)
	}
}

func TestGenerate_MissingRequiredField(t *testing.T) {
	resetGenerateFlags(t)
	dir := t.TempDir()
	content := "---\nextends: substitution\n---\n\n# No message\n\n```vale-swap\nfoo: bar\n```\n"
	mdPath := filepath.Join(dir, "bad.md")
	os.WriteFile(mdPath, []byte(content), 0644)

	rootCmd.SetArgs([]string{"generate", mdPath})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing message")
	}
}
