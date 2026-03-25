package hugo

import (
	"testing"
)

func TestParseAndCheckVersion_Valid(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantVer string
	}{
		{
			name:    "extended with deploy",
			output:  "hugo v0.159.0+extended+withdeploy darwin/arm64 BuildDate=2026-03-23T18:16:59Z VendorInfo=Homebrew",
			wantVer: "0.159.0",
		},
		{
			name:    "plain version",
			output:  "hugo v0.128.0 linux/amd64 BuildDate=2024-06-01",
			wantVer: "0.128.0",
		},
		{
			name:    "extended only",
			output:  "hugo v0.140.2+extended darwin/arm64 BuildDate=unknown",
			wantVer: "0.140.2",
		},
		{
			name:    "minimum version exactly",
			output:  "hugo v0.128.0 whatever",
			wantVer: "0.128.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, err := ParseAndCheckVersion(tt.output)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ver != tt.wantVer {
				t.Fatalf("got version %q, want %q", ver, tt.wantVer)
			}
		})
	}
}

func TestParseAndCheckVersion_TooOld(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "old version",
			output: "hugo v0.100.0 linux/amd64",
		},
		{
			name:   "slightly below minimum",
			output: "hugo v0.127.9 darwin/arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAndCheckVersion(tt.output)
			if err == nil {
				t.Fatal("expected error for old Hugo version, got nil")
			}
			be, ok := err.(*BuildError)
			if !ok {
				t.Fatalf("expected *BuildError, got %T", err)
			}
			if be.ExitCode != 3 {
				t.Fatalf("expected exit code 3, got %d", be.ExitCode)
			}
		})
	}
}

func TestParseAndCheckVersion_Unparseable(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "no version",
			output: "hugo is great",
		},
		{
			name:   "empty output",
			output: "",
		},
		{
			name:   "missing v prefix",
			output: "hugo 0.159.0 darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAndCheckVersion(tt.output)
			if err == nil {
				t.Fatal("expected error for unparseable output, got nil")
			}
			be, ok := err.(*BuildError)
			if !ok {
				t.Fatalf("expected *BuildError, got %T", err)
			}
			if be.ExitCode != 3 {
				t.Fatalf("expected exit code 3, got %d", be.ExitCode)
			}
		})
	}
}

func TestFindHugo_EmptyPath(t *testing.T) {
	// This test verifies that FindHugo with an empty path falls back to $PATH.
	// It will succeed if hugo is installed, and fail with a *BuildError if not.
	path, err := FindHugo("")
	if err != nil {
		be, ok := err.(*BuildError)
		if !ok {
			t.Fatalf("expected *BuildError, got %T: %v", err, err)
		}
		if be.ExitCode != 3 {
			t.Fatalf("expected exit code 3, got %d", be.ExitCode)
		}
		t.Skipf("hugo not found on $PATH, skipping: %v", err)
	}
	if path == "" {
		t.Fatal("FindHugo returned empty path with no error")
	}
}

func TestFindHugo_InvalidPath(t *testing.T) {
	_, err := FindHugo("/nonexistent/path/to/hugo")
	if err == nil {
		t.Fatal("expected error for invalid Hugo path")
	}
	be, ok := err.(*BuildError)
	if !ok {
		t.Fatalf("expected *BuildError, got %T", err)
	}
	if be.ExitCode != 3 {
		t.Fatalf("expected exit code 3, got %d", be.ExitCode)
	}
}

func TestBuildError_Error(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		err := &BuildError{
			ExitCode: 3,
			Message:  "test message",
			Err:      nil,
		}
		if err.Error() != "test message" {
			t.Fatalf("got %q, want %q", err.Error(), "test message")
		}
	})

	t.Run("with message only", func(t *testing.T) {
		inner := &BuildError{ExitCode: 4, Message: "inner"}
		err := &BuildError{
			ExitCode: 3,
			Message:  "outer",
			Err:      inner,
		}
		want := "outer: inner"
		if err.Error() != want {
			t.Fatalf("got %q, want %q", err.Error(), want)
		}
	})
}
