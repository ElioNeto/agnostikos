package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// installAgnosticBinary
// ---------------------------------------------------------------------------

func TestInstallAgnosticBinary_EmptyRootfsDir(t *testing.T) {
	err := installAgnosticBinary("", "amd64")
	if err == nil {
		t.Fatal("expected error for empty rootfsDir")
	}
	if !strings.Contains(err.Error(), "rootfsDir must not be empty") {
		t.Errorf("expected 'rootfsDir must not be empty' error, got: %v", err)
	}
}

func TestInstallAgnosticBinary_CreatesDirectoriesAndSymlink(t *testing.T) {
	tmp := t.TempDir()

	// In the test environment, os.Executable() returns the test binary,
	// so the "running" strategy is used. This copies the test binary,
	// which is a valid Go binary, into the rootfs.
	err := installAgnosticBinary(tmp, "amd64")
	if err != nil {
		t.Fatalf("installAgnosticBinary failed: %v", err)
	}

	// Verify binary installed at /usr/bin/agnostic
	binaryPath := filepath.Join(tmp, "usr", "bin", "agnostic")
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("expected binary at %s: %v", binaryPath, err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("binary permissions: got %o, want 0755", info.Mode().Perm())
	}

	// Verify symlink at /usr/local/bin/agnostic -> /usr/bin/agnostic
	symlinkPath := filepath.Join(tmp, "usr", "local", "bin", "agnostic")
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("expected symlink at %s: %v", symlinkPath, err)
	}
	if target != "/usr/bin/agnostic" {
		t.Errorf("symlink target: got %q, want %q", target, "/usr/bin/agnostic")
	}
}

func TestInstallAgnosticBinary_DefaultArch(t *testing.T) {
	tmp := t.TempDir()

	err := installAgnosticBinary(tmp, "")
	if err != nil {
		t.Fatalf("installAgnosticBinary with empty arch failed: %v", err)
	}

	// Binary should exist (arch defaults to runtime.GOARCH)
	binaryPath := filepath.Join(tmp, "usr", "bin", "agnostic")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Errorf("expected binary at %s with default arch", binaryPath)
	}

	// Symlink should exist
	symlinkPath := filepath.Join(tmp, "usr", "local", "bin", "agnostic")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Errorf("expected symlink at %s", symlinkPath)
	}
}

func TestInstallAgnosticBinary_BinaryIsExecutable(t *testing.T) {
	tmp := t.TempDir()

	err := installAgnosticBinary(tmp, "amd64")
	if err != nil {
		t.Fatalf("installAgnosticBinary failed: %v", err)
	}

	binaryPath := filepath.Join(tmp, "usr", "bin", "agnostic")
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("failed to read binary: %v", err)
	}

	// Binary should be non-empty (it was copied from the test binary)
	if len(data) == 0 {
		t.Error("binary is empty")
	}

	// Verify it's an ELF binary (starts with \x7fELF)
	if len(data) < 4 || data[0] != 0x7f || data[1] != 'E' || data[2] != 'L' || data[3] != 'F' {
		t.Error("binary does not appear to be an ELF file")
	}
}

// ---------------------------------------------------------------------------
// findModuleRoot
// ---------------------------------------------------------------------------

func TestFindModuleRoot(t *testing.T) {
	t.Run("from current directory", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		root := findModuleRoot(cwd)
		if root == "" {
			t.Fatal("findModuleRoot returned empty from cwd")
		}
		// Should contain go.mod
		if _, err := os.Stat(filepath.Join(root, "go.mod")); os.IsNotExist(err) {
			t.Errorf("module root %s does not contain go.mod", root)
		}
	})

	t.Run("from non-existent directory returns empty", func(t *testing.T) {
		root := findModuleRoot("/nonexistent/path/that/does/not/exist")
		if root != "" {
			t.Errorf("expected empty for non-existent dir, got %q", root)
		}
	})

	t.Run("from subdirectory finds root", func(t *testing.T) {
		tmp := t.TempDir()
		subdir := filepath.Join(tmp, "a", "b", "c")
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatal(err)
		}

		// No go.mod in hierarchy -> empty
		root := findModuleRoot(subdir)
		if root != "" {
			t.Errorf("expected empty when no go.mod exists, got %q", root)
		}

		// Create go.mod at tmp
		if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatal(err)
		}

		root = findModuleRoot(subdir)
		if root == "" {
			t.Fatal("expected module root to be found")
		}
		if root != tmp {
			t.Errorf("expected module root %q, got %q", tmp, root)
		}
	})
}

func TestFindModuleRoot_NoGoMod(t *testing.T) {
	tmp := t.TempDir()
	root := findModuleRoot(tmp)
	if root != "" {
		t.Errorf("expected empty when no go.mod exists, got %q", root)
	}
}

// ---------------------------------------------------------------------------
// copyFile
// ---------------------------------------------------------------------------

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	content := "hello world"
	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(dst, src); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("copied content = %q, want %q", string(data), content)
	}
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	tmp := t.TempDir()
	err := copyFile(filepath.Join(tmp, "dst.txt"), filepath.Join(tmp, "nonexistent.txt"))
	if err == nil {
		t.Fatal("expected error when source does not exist")
	}
}

// ---------------------------------------------------------------------------
// findAgnosticBinary (with osExecutable mocked)
// ---------------------------------------------------------------------------

func TestFindAgnosticBinary_DistStrategy(t *testing.T) {
	tmp := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldCwd) }()

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	// Create a fake dist binary
	distDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}
	distBinary := filepath.Join(distDir, "agnostic-amd64")
	if err := os.WriteFile(distBinary, []byte("fake binary"), 0644); err != nil {
		t.Fatal(err)
	}

	// Mock osExecutable to return a non-matching binary so "running" is skipped
	origExec := osExecutable
	osExecutable = func() (string, error) { return "/usr/bin/some-other-binary", nil }
	t.Cleanup(func() { osExecutable = origExec })

	source, strategy, err := findAgnosticBinary("amd64")
	if err != nil {
		t.Fatalf("findAgnosticBinary failed: %v", err)
	}
	if strategy != "dist" {
		t.Errorf("expected strategy 'dist', got %q", strategy)
	}
	if source != distBinary {
		t.Errorf("expected source %q, got %q", distBinary, source)
	}
}

func TestFindAgnosticBinary_DistStrategyWithExecutableError(t *testing.T) {
	tmp := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldCwd) }()

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	// Create a fake dist binary
	distDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}
	distBinary := filepath.Join(distDir, "agnostic-amd64")
	if err := os.WriteFile(distBinary, []byte("fake binary"), 0644); err != nil {
		t.Fatal(err)
	}

	// Mock osExecutable to return an error
	origExec := osExecutable
	osExecutable = func() (string, error) { return "", errors.New("exec error") }
	t.Cleanup(func() { osExecutable = origExec })

	source, strategy, err := findAgnosticBinary("amd64")
	if err != nil {
		t.Fatalf("findAgnosticBinary failed: %v", err)
	}
	if strategy != "dist" {
		t.Errorf("expected strategy 'dist', got %q", strategy)
	}
	if source != distBinary {
		t.Errorf("expected source %q, got %q", distBinary, source)
	}
}

func TestFindAgnosticBinary_BuildStrategy(t *testing.T) {
	tmp := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldCwd) }()

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	// Create go.mod in tmp so it becomes the module root
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Mock osExecutable to return a non-matching binary so "running" is skipped
	origExec := osExecutable
	osExecutable = func() (string, error) { return "/usr/bin/some-other-binary", nil }
	t.Cleanup(func() { osExecutable = origExec })

	source, strategy, err := findAgnosticBinary("arm64")
	if err != nil {
		t.Fatalf("findAgnosticBinary failed: %v", err)
	}
	if strategy != "build" {
		t.Errorf("expected strategy 'build', got %q", strategy)
	}
	if source != tmp {
		t.Errorf("expected source %q, got %q", tmp, source)
	}
}

func TestFindAgnosticBinary_NoSource(t *testing.T) {
	tmp := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldCwd) }()

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	// No go.mod, no dist binary, and osExecutable returns a non-matching binary
	origExec := osExecutable
	osExecutable = func() (string, error) { return "/usr/bin/some-other-binary", nil }
	t.Cleanup(func() { osExecutable = origExec })

	_, _, err = findAgnosticBinary("amd64")
	if err == nil {
		t.Fatal("expected error when no source available")
	}
	if !strings.Contains(err.Error(), "no source found") {
		t.Errorf("expected 'no source found' error, got: %v", err)
	}
}

func TestFindAgnosticBinary_RunningStrategy(t *testing.T) {
	// Without mocking osExecutable, the test binary itself is found
	// (since it has .test suffix, which matches our "running" check).
	source, strategy, err := findAgnosticBinary("amd64")
	if err != nil {
		t.Fatalf("findAgnosticBinary failed: %v", err)
	}
	if strategy != "running" {
		t.Errorf("expected strategy 'running', got %q", strategy)
	}
	if source == "" {
		t.Error("expected non-empty source for running strategy")
	}
}

// ---------------------------------------------------------------------------
// buildBinary
// ---------------------------------------------------------------------------

func TestBuildBinary(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "agnostic")

	// Find the actual module root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := findModuleRoot(cwd)
	if moduleRoot == "" {
		// If we can't find the module root, try the parent of cwd
		moduleRoot = filepath.Dir(cwd)
		if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); os.IsNotExist(err) {
			t.Skip("not running from project module root, skipping build test")
		}
	}

	err = buildBinary(dest, moduleRoot)
	if err != nil {
		t.Fatalf("buildBinary failed: %v", err)
	}

	// Verify binary was created and is an ELF
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("expected binary at %s: %v", dest, err)
	}
	if info.Size() == 0 {
		t.Error("built binary is empty")
	}
}

func TestBuildBinary_InvalidModuleRoot(t *testing.T) {
	tmp := t.TempDir()
	err := buildBinary(filepath.Join(tmp, "agnostic"), "/nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid module root")
	}
}

// ---------------------------------------------------------------------------
// Error paths
// ---------------------------------------------------------------------------

func TestInstallAgnosticBinary_RejectsInvalidDest(t *testing.T) {
	// rootfsDir that cannot be created (e.g., deeply nested under a file)
	tmp := t.TempDir()

	// Create a file at usr, so usr/bin can't be created
	usrPath := filepath.Join(tmp, "usr")
	if err := os.WriteFile(usrPath, []byte("not-a-directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := installAgnosticBinary(tmp, "amd64")
	if err == nil {
		t.Fatal("expected error when usr is a file")
	}
}
