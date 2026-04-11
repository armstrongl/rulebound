package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestYML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test yml: %v", err)
	}
	return path
}

func TestValidate_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestYML(t, dir, "valid.yml", `extends: substitution
message: "test"
level: warning
swap:
  foo: bar
`)

	rootCmd.SetArgs([]string{"validate", path})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_MultipleValidFiles(t *testing.T) {
	dir := t.TempDir()
	path1 := writeTestYML(t, dir, "a.yml", `extends: existence
message: "test"
level: warning
tokens:
  - foo
`)
	path2 := writeTestYML(t, dir, "b.yml", `extends: occurrence
message: "test"
level: warning
max: 10
token: '\S+'
`)

	rootCmd.SetArgs([]string{"validate", path1, path2})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestYML(t, dir, "bad.yml", `extends: substitution
level: warning
`)

	rootCmd.SetArgs([]string{"validate", path})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid file")
	}
}

func TestValidate_MixedFiles(t *testing.T) {
	dir := t.TempDir()
	good := writeTestYML(t, dir, "good.yml", `extends: existence
message: "test"
level: warning
tokens:
  - foo
`)
	bad := writeTestYML(t, dir, "bad.yml", `extends: substitution
level: warning
`)

	rootCmd.SetArgs([]string{"validate", good, bad})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when one file is invalid")
	}
}

func TestValidate_FileNotExist(t *testing.T) {
	rootCmd.SetArgs([]string{"validate", "/nonexistent/file.yml"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestValidate_GenerateThenValidate(t *testing.T) {
	// Integration: generate YAML from .md, then validate the output.
	resetGenerateFlags(t)
	dir := t.TempDir()
	mdPath := writeTestMD(t, dir, "Integration.md")
	ymlPath := filepath.Join(dir, "Integration.yml")

	// Generate.
	rootCmd.SetArgs([]string{"generate", mdPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	// Validate the generated output.
	rootCmd.SetArgs([]string{"validate", ymlPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("validate failed on generated file: %v", err)
	}

	// Verify the generated file contains expected content.
	data, err := os.ReadFile(ymlPath)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}
	if !strings.Contains(string(data), "extends: substitution") {
		t.Error("generated file should contain extends: substitution")
	}
}
