package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// installAgnosticBinary installs the agnostic binary into the rootfs.
// Strategy (in order of precedence):
//  1. Copy the currently running binary (os.Executable())
//  2. Look for pre-compiled binary at dist/agnostic-<arch>
//  3. Compile on-the-fly via go build if source is available
//
// The binary is always built with CGO_ENABLED=0 for static linking,
// and installed with mode 0755 at <rootfsDir>/usr/bin/agnostic.
// A symlink is created at <rootfsDir>/usr/local/bin/agnostic -> /usr/bin/agnostic.
func installAgnosticBinary(rootfsDir, arch string) error {
	if rootfsDir == "" {
		return fmt.Errorf("rootfsDir must not be empty")
	}
	if arch == "" {
		arch = runtime.GOARCH
	}

	destDir := filepath.Join(rootfsDir, "usr", "bin")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, "agnostic")

	source, strategy, err := findAgnosticBinary(arch)
	if err != nil {
		return fmt.Errorf("find agnostic binary: %w", err)
	}

	fmt.Printf("[install-binary] using strategy %q: %s\n", strategy, source)

	// Copy or compile the binary
	switch strategy {
	case "running", "dist":
		if err := copyFile(destPath, source); err != nil {
			return fmt.Errorf("copy binary from %s: %w", source, err)
		}
	case "build":
		if err := buildBinary(destPath, source); err != nil {
			return fmt.Errorf("build binary: %w", err)
		}
	default:
		return fmt.Errorf("unknown strategy %q", strategy)
	}

	// Ensure executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("chmod %s: %w", destPath, err)
	}

	// Create symlink at /usr/local/bin/agnostic -> /usr/bin/agnostic
	localBinDir := filepath.Join(rootfsDir, "usr", "local", "bin")
	if err := os.MkdirAll(localBinDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", localBinDir, err)
	}

	symlinkPath := filepath.Join(localBinDir, "agnostic")
	_ = os.Remove(symlinkPath) // Remove existing file/symlink if any
	if err := os.Symlink("/usr/bin/agnostic", symlinkPath); err != nil {
		return fmt.Errorf("symlink %s -> /usr/bin/agnostic: %w", symlinkPath, err)
	}

	fmt.Printf("[install-binary] installed agnostic binary at %s\n", destPath)
	fmt.Printf("[install-binary] created symlink %s -> /usr/bin/agnostic\n", symlinkPath)

	return nil
}

// Overrideable for tests — same pattern as httpClient in rootfs.go.
var osExecutable = os.Executable

// findAgnosticBinary locates the agnostic binary using the strategy precedence:
// 1. Running binary (os.Executable())
// 2. Pre-compiled binary at dist/agnostic-<arch>
// 3. Go source available (go.mod exists in the module root)
//
// Returns the source path, the strategy name, and any error.
func findAgnosticBinary(arch string) (source string, strategy string, err error) {
	// Strategy 1: currently running binary
	execPath, execErr := osExecutable()
	if execErr == nil && execPath != "" {
		// Verify it's actually our binary by checking the name
		base := filepath.Base(execPath)
		if base == "agnostic" || base == "agnostic.exe" || base == "agnostic.test" {
			return execPath, "running", nil
		}
		// Also check if it looks like a go-built test binary
		if strings.HasSuffix(base, ".test") {
			return execPath, "running", nil
		}
	}

	// Strategy 2: pre-compiled binary in dist/
	distName := fmt.Sprintf("dist/agnostic-%s", arch)
	if _, statErr := os.Stat(distName); statErr == nil {
		absPath, absErr := filepath.Abs(distName)
		if absErr == nil {
			return absPath, "dist", nil
		}
		return distName, "dist", nil
	}

	// Strategy 3: compile on-the-fly
	// Check if go.mod exists in the module root (source available)
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		return "", "", fmt.Errorf("getwd: %w", cwdErr)
	}

	// Walk up to find go.mod
	moduleRoot := findModuleRoot(cwd)
	if moduleRoot == "" {
		return "", "", fmt.Errorf("no source found: running binary not available, no dist binary at %q, and no go.mod found from %s", distName, cwd)
	}

	return moduleRoot, "build", nil
}

// findModuleRoot walks up from dir looking for go.mod.
// Returns the directory containing go.mod, or empty string if not found.
func findModuleRoot(dir string) string {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return ""
		}
		dir = parent
	}
}

// buildBinary compiles the agnostic binary from source at moduleRoot.
// It runs `go build` with CGO_ENABLED=0 for a static binary.
func buildBinary(destPath, moduleRoot string) error {
	buildCmd := exec.Command("go", "build", "-o", destPath, ".")
	buildCmd.Dir = moduleRoot
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, string(output))
	}

	return nil
}

// copyFile copies a file from src to dst using os primitives.
// The destination file is created with 0644 mode (then chmodded to 0755 by the caller).
func copyFile(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write destination %s: %w", dst, err)
	}
	return nil
}
