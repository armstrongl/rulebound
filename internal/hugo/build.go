package hugo

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// minHugoVersion is the minimum Hugo version required to build.
const minHugoVersion = "0.128.0"

// BuildError wraps a Hugo error with an exit code for the CLI layer.
type BuildError struct {
	ExitCode int
	Message  string
	Err      error
}

func (e *BuildError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *BuildError) Unwrap() error {
	return e.Err
}

// hugoVersionRegex extracts the semver portion from `hugo version` output.
// For example, "hugo v0.159.0+extended+withdeploy darwin/arm64 ..." yields "0.159.0".
var hugoVersionRegex = regexp.MustCompile(`v(\d+\.\d+\.\d+)`)

// FindHugo locates the Hugo binary. If hugoPath is non-empty, FindHugo uses
// it directly; otherwise it looks up Hugo on $PATH.
// It returns the resolved path, or a *BuildError with exit code 3 on failure.
func FindHugo(hugoPath string) (string, error) {
	if hugoPath != "" {
		// Verify that the explicit path exists and is executable.
		path, err := exec.LookPath(hugoPath)
		if err != nil {
			return "", &BuildError{
				ExitCode: 3,
				Message:  fmt.Sprintf("hugo binary not found at %s", hugoPath),
				Err:      err,
			}
		}
		return path, nil
	}

	path, err := exec.LookPath("hugo")
	if err != nil {
		return "", &BuildError{
			ExitCode: 3,
			Message:  "hugo binary not found on $PATH; install Hugo or use --hugo flag",
			Err:      err,
		}
	}
	return path, nil
}

// CheckHugoVersion runs `hugo version` and verifies that the version is at least minHugoVersion.
// It returns the parsed version string, or a *BuildError with exit code 3 on failure.
func CheckHugoVersion(hugoBin string) (string, error) {
	cmd := exec.Command(hugoBin, "version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &bytes.Buffer{}

	if err := cmd.Run(); err != nil {
		return "", &BuildError{
			ExitCode: 3,
			Message:  "failed to run hugo version",
			Err:      err,
		}
	}

	return ParseAndCheckVersion(stdout.String())
}

// ParseAndCheckVersion extracts and validates a Hugo version string from
// `hugo version` output. It is exported for use in tests.
func ParseAndCheckVersion(output string) (string, error) {
	matches := hugoVersionRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		return "", &BuildError{
			ExitCode: 3,
			Message:  fmt.Sprintf("could not parse Hugo version from output: %s", strings.TrimSpace(output)),
		}
	}

	versionStr := matches[1]
	ver, err := semver.NewVersion(versionStr)
	if err != nil {
		return "", &BuildError{
			ExitCode: 3,
			Message:  fmt.Sprintf("invalid Hugo version %q", versionStr),
			Err:      err,
		}
	}

	minVer, err := semver.NewVersion(minHugoVersion)
	if err != nil {
		return "", &BuildError{
			ExitCode: 3,
			Message:  fmt.Sprintf("invalid minimum version constraint %q", minHugoVersion),
			Err:      err,
		}
	}

	if ver.LessThan(minVer) {
		return "", &BuildError{
			ExitCode: 3,
			Message:  fmt.Sprintf("Hugo version %s is too old; minimum required is %s", versionStr, minHugoVersion),
		}
	}

	return versionStr, nil
}

// BuildResult holds the captured output of a Hugo build invocation.
type BuildResult struct {
	// Stdout is the captured Hugo stdout output.
	Stdout string
	// Stderr is the captured Hugo stderr output.
	Stderr string
}

// Build runs `hugo build --source <sourceDir> --destination <destDir>`.
// It returns a *BuildError with exit code 4 if Hugo exits non-zero.
func Build(hugoBin, sourceDir, destDir string) (*BuildResult, error) {
	cmd := exec.Command(hugoBin, "build", "--source", sourceDir, "--destination", destDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &BuildResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		return result, &BuildError{
			ExitCode: 4,
			Message:  fmt.Sprintf("hugo build failed: %s", strings.TrimSpace(stderr.String())),
			Err:      err,
		}
	}

	return result, nil
}

// RunPagefind runs `pagefind --site <siteDir>` as a post-build step.
// If pagefind is not on $PATH, RunPagefind returns (false, nil) so the caller
// can degrade gracefully. If pagefind runs successfully, it returns (true, nil).
// On execution failure, it returns (true, error).
func RunPagefind(siteDir string) (found bool, err error) {
	pagefindBin, lookErr := exec.LookPath("pagefind")
	if lookErr != nil {
		return false, nil
	}

	cmd := exec.Command(pagefindBin, "--site", siteDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return true, fmt.Errorf("pagefind failed: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	return true, nil
}
