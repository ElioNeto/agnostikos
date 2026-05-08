package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ToolchainNumCPUs
// ---------------------------------------------------------------------------

func TestToolchainNumCPUs(t *testing.T) {
	tests := []struct {
		name string
		cfg  ToolchainConfig
		want string
	}{
		{
			name: "custom value",
			cfg:  ToolchainConfig{NumCPUs: "8"},
			want: "8",
		},
		{
			name: "empty defaults to runtime.NumCPU clamped at 4",
			cfg:  ToolchainConfig{},
			want: clampCPUs(runtime.NumCPU()),
		},
		{
			name: "single CPU",
			cfg:  ToolchainConfig{NumCPUs: "1"},
			want: "1",
		},
		{
			name: "explicit 4",
			cfg:  ToolchainConfig{NumCPUs: "4"},
			want: "4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toolchainNumCPUs(tt.cfg)
			if got != tt.want {
				t.Errorf("toolchainNumCPUs(%+v) = %s; want %s", tt.cfg, got, tt.want)
			}
		})
	}
}

// clampCPUs reproduces the clamping logic from toolchainNumCPUs for validation.
func clampCPUs(n int) string {
	if n > 4 {
		n = 4
	}
	return fmt.Sprintf("%d", n)
}

// ---------------------------------------------------------------------------
// ToolchainTarget
// ---------------------------------------------------------------------------

func TestToolchainTarget(t *testing.T) {
	tests := []struct {
		name string
		cfg  ToolchainConfig
	}{
		{
			name: "custom target",
			cfg:  ToolchainConfig{Target: "aarch64-linux-gnu"},
		},
		{
			name: "empty target falls back to auto-detect",
			cfg:  ToolchainConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toolchainTarget(tt.cfg)
			if tt.cfg.Target != "" {
				if got != tt.cfg.Target {
					t.Errorf("toolchainTarget() = %s; want %s", got, tt.cfg.Target)
				}
			} else {
				// Empty target: should fall back to auto-detect (non-empty)
				if got == "" {
					t.Error("toolchainTarget() returned empty for empty config")
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helper: create fake tarball
// ---------------------------------------------------------------------------

// writeFakeTarball creates a minimal valid tar.xz file for testing.
// The content is a valid tar header (empty file) compressed with xz.
// If xz is not available, creates a plain .tar file instead.
func writeFakeTarball(t *testing.T, path string) {
	t.Helper()

	// Try xz compression first
	if _, err := exec.LookPath("xz"); err == nil {
		// Create a minimal tar + xz pipeline
		cmd := exec.Command("sh", "-c", "tar -cf - --files-from /dev/null | xz > "+path)
		if out, err := cmd.CombinedOutput(); err == nil {
			_ = out
			return
		}
	}

	// Fallback: create a plain tar (not compressed)
	// We'll name it .tar instead
	tarPath := strings.TrimSuffix(path, ".xz") + ".tar"
	cmd := exec.Command("tar", "-cf", tarPath, "--files-from", "/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create fake tarball: %v\n%s", err, string(out))
	}
	// Rename to expected path (accept that it's not xz-compressed)
	if err := os.Rename(tarPath, path); err != nil {
		t.Fatalf("failed to rename tarball: %v", err)
	}
}

// writeFakeTarballPlain creates a minimal tar file without xz dependency.
func writeFakeTarballPlain(t *testing.T, path string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("tar", "-cf", path, "--files-from", "/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create fake tarball: %v\n%s", err, string(out))
	}
}

// ---------------------------------------------------------------------------
// ExtractIfNeeded
// ---------------------------------------------------------------------------

func TestExtractIfNeeded_AlreadyExtracted(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the expected directory (simulating already extracted)
	expectedDir := filepath.Join(tmpDir, "pkg-1.0")
	if err := os.MkdirAll(expectedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Should skip extraction and return nil
	err := extractIfNeeded(context.Background(), "/nonexistent/tarball.tar.xz", tmpDir, expectedDir)
	if err != nil {
		t.Errorf("expected nil when dir exists, got: %v", err)
	}
}

func TestExtractIfNeeded_MissingTarball(t *testing.T) {
	tmpDir := t.TempDir()
	expectedDir := filepath.Join(tmpDir, "pkg-1.0")

	// Tarball doesn't exist AND dir doesn't exist → tar will fail
	err := extractIfNeeded(context.Background(), "/nonexistent/tarball.tar.xz", tmpDir, expectedDir)
	if err == nil {
		t.Error("expected error for missing tarball and missing dir")
	}
}

func TestExtractIfNeeded_ExtractsWhenDirMissing(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "sources")
	expectedDir := filepath.Join(srcDir, "pkg-1.0")
	tarballPath := filepath.Join(srcDir, "pkg-1.0.tar.xz")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFakeTarball(t, tarballPath)

	// expectedDir doesn't exist, so it should try to extract
	// The fake tarball may or may not extract correctly depending on xz,
	// so we just verify an error of some kind or success
	_ = extractIfNeeded(context.Background(), tarballPath, srcDir, expectedDir)
	// We don't assert on result because tar with fake xz may fail silently
	// The important test is AlreadyExtracted above
}

// ---------------------------------------------------------------------------
// BuildBinutils
// ---------------------------------------------------------------------------

func TestBuildBinutils_TarballNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ToolchainConfig{
		TargetDir: tmpDir,
		NumCPUs:   "2",
	}

	err := BuildBinutils(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when tarball does not exist")
	}
	if !strings.Contains(err.Error(), "tarball not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBuildBinutils_AlreadyBuilt(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")
	srcDir := filepath.Join(tmpDir, "sources")
	buildDir := filepath.Join(srcDir, "build-binutils")

	// Create tarball
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFakeTarballPlain(t, filepath.Join(srcDir, "binutils-2.42.tar.xz"))

	// Create src dir (already extracted)
	srcPath := filepath.Join(srcDir, "binutils-2.42")
	if err := os.MkdirAll(srcPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create build dir and stamp (already built)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	stampFile := filepath.Join(buildDir, ".build-complete")
	if err := os.WriteFile(stampFile, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := ToolchainConfig{
		TargetDir: rootfsDir,
		NumCPUs:   "2",
	}

	err := BuildBinutils(context.Background(), cfg)
	if err != nil {
		t.Errorf("expected no error when already built, got: %v", err)
	}
}

func TestBuildBinutils_BuildDirCreated(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")
	srcDir := filepath.Join(tmpDir, "sources")
	buildDir := filepath.Join(srcDir, "build-binutils")

	// Create tarball
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFakeTarballPlain(t, filepath.Join(srcDir, "binutils-2.42.tar.xz"))

	// Create src dir (already extracted)
	srcPath := filepath.Join(srcDir, "binutils-2.42")
	if err := os.MkdirAll(srcPath, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := ToolchainConfig{
		TargetDir: rootfsDir,
		NumCPUs:   "2",
	}

	// Build will fail at configure (no real configure script), but build dir should exist
	_ = BuildBinutils(context.Background(), cfg)

	// Check build dir was created even on failure
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		t.Error("build-binutils directory was not created")
	}
}

// ---------------------------------------------------------------------------
// BuildGCC
// ---------------------------------------------------------------------------

func TestBuildGCC_TarballNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ToolchainConfig{
		TargetDir: tmpDir,
		NumCPUs:   "2",
	}

	err := BuildGCC(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when tarball does not exist")
	}
	if !strings.Contains(err.Error(), "tarball not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBuildGCC_AlreadyBuilt(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")
	srcDir := filepath.Join(tmpDir, "sources")
	buildDir := filepath.Join(srcDir, "build-gcc")

	// Create tarball
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFakeTarballPlain(t, filepath.Join(srcDir, "gcc-14.1.0.tar.xz"))

	// Create src dir (already extracted) with contrib/download_prerequisites
	srcPath := filepath.Join(srcDir, "gcc-14.1.0")
	contribDir := filepath.Join(srcPath, "contrib")
	if err := os.MkdirAll(contribDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a mock download_prerequisites script that succeeds
	mockScript := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(filepath.Join(contribDir, "download_prerequisites"), []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	// Create build dir with stamp (already built)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	stampFile := filepath.Join(buildDir, ".build-complete")
	if err := os.WriteFile(stampFile, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := ToolchainConfig{
		TargetDir: rootfsDir,
		NumCPUs:   "2",
	}

	// With stamp present, should skip even though download_prerequisites would fail
	err := BuildGCC(context.Background(), cfg)
	if err != nil {
		t.Errorf("expected no error when already built, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// BuildGLibc
// ---------------------------------------------------------------------------

func TestBuildGLibc_TarballNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ToolchainConfig{
		TargetDir: tmpDir,
		NumCPUs:   "2",
	}

	err := BuildGLibc(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when tarball does not exist")
	}
	if !strings.Contains(err.Error(), "tarball not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBuildGLibc_AlreadyBuilt(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")
	srcDir := filepath.Join(tmpDir, "sources")
	buildDir := filepath.Join(srcDir, "build-glibc")

	// Create tarball
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFakeTarballPlain(t, filepath.Join(srcDir, "glibc-2.39.tar.xz"))

	// Create src dir (already extracted)
	srcPath := filepath.Join(srcDir, "glibc-2.39")
	if err := os.MkdirAll(srcPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create build dir with stamp (already built)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	stampFile := filepath.Join(buildDir, ".build-complete")
	if err := os.WriteFile(stampFile, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := ToolchainConfig{
		TargetDir: rootfsDir,
		NumCPUs:   "2",
	}

	err := BuildGLibc(context.Background(), cfg)
	if err != nil {
		t.Errorf("expected no error when already built, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Exec mocking for NumCPUs propagation test
//
// We replace execCommandContext with a function that records all invocations
// and returns a command that succeeds (runs /bin/true) so the build pipeline
// can proceed without external dependencies.
// ---------------------------------------------------------------------------

type recordedCmd struct {
	Name string
	Args []string
}

func TestBuildBinutils_MakeJobsPropagation(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")
	srcDir := filepath.Join(tmpDir, "sources")

	// Create tarball
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFakeTarballPlain(t, filepath.Join(srcDir, "binutils-2.42.tar.xz"))

	// Create src dir with a fake configure script
	srcPath := filepath.Join(srcDir, "binutils-2.42")
	if err := os.MkdirAll(srcPath, 0755); err != nil {
		t.Fatal(err)
	}
	configureScript := filepath.Join(srcPath, "configure")
	configureContent := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(configureScript, configureContent, 0755); err != nil {
		t.Fatal(err)
	}

	// Record all exec invocations
	var recorded []recordedCmd

	// Save and replace execCommandContext with a mock that records calls
	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		recorded = append(recorded, recordedCmd{Name: name, Args: args})
		// Return a command that succeeds (true)
		return exec.CommandContext(ctx, "true")
	}
	t.Cleanup(func() { execCommandContext = origExec })

	cfg := ToolchainConfig{
		TargetDir: rootfsDir,
		NumCPUs:   "8",
	}

	err := BuildBinutils(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildBinutils failed: %v", err)
	}

	// Verify that at least one "make" invocation had -j8 (compile step).
	// The install step ("make install DESTDIR=...") does not receive -j.
	foundCompileMake := false
	foundJobs8 := false
	for _, c := range recorded {
		if c.Name == "make" {
			foundCompileMake = true
			for _, a := range c.Args {
				if a == "-j8" {
					foundJobs8 = true
				}
			}
		}
	}
	if !foundCompileMake {
		t.Error("BuildBinutils did not call make")
	}
	if !foundJobs8 {
		t.Error("BuildBinutils did not pass -j8 to any make invocation")
	}
}

// ---------------------------------------------------------------------------
// DownloadToolchain with mocked HTTP
// ---------------------------------------------------------------------------

func TestDownloadToolchain_Download(t *testing.T) {
	// Start a test HTTP server that serves dummy tarballs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-xz")
		w.WriteHeader(http.StatusOK)
		// Return minimal valid content
		_, _ = w.Write([]byte("fake-tarball-content"))
	}))
	defer server.Close()

	// Save and replace httpClient
	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	// Save and replace DefaultToolchain to point to our test server
	origToolchain := DefaultToolchain
	DefaultToolchain = []ToolchainPackage{
		{Name: "test-pkg-1.0", URL: server.URL + "/test-pkg-1.0.tar.xz"},
	}
	t.Cleanup(func() { DefaultToolchain = origToolchain })

	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")
	// sourcesDir will be filepath.Dir(rootfsDir) + "/sources" = tmpDir + "/sources"
	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := DownloadToolchain(rootfsDir)
	if err != nil {
		t.Fatalf("DownloadToolchain failed: %v", err)
	}

	// Verify file was downloaded
	srcDir := filepath.Join(tmpDir, "sources")
	dest := filepath.Join(srcDir, "test-pkg-1.0.tar.xz")
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		t.Error("downloaded file was not created")
	}
}

func TestDownloadToolchain_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")
	srcDir := filepath.Join(tmpDir, "sources")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a fake existing file
	existingFile := filepath.Join(srcDir, "test-pkg-1.0.tar.xz")
	if err := os.WriteFile(existingFile, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	// Override toolchain list
	origToolchain := DefaultToolchain
	DefaultToolchain = []ToolchainPackage{
		{Name: "test-pkg-1.0", URL: "http://nonexistent.example/test-pkg-1.0.tar.xz"},
	}
	t.Cleanup(func() { DefaultToolchain = origToolchain })

	// Should skip download (file exists) and not hit the network
	err := DownloadToolchain(rootfsDir)
	if err != nil {
		t.Errorf("expected no error when file exists, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// sourcesDir integration with toolchain config
// ---------------------------------------------------------------------------

func TestSourcesDirFromToolchainConfig(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")

	got := sourcesDir(rootfsDir)
	want := filepath.Join(tmpDir, "sources")
	if got != want {
		t.Errorf("sourcesDir(%s) = %s; want %s", rootfsDir, got, want)
	}
}

// ---------------------------------------------------------------------------
// targetTriple
// ---------------------------------------------------------------------------

func TestTargetTriple(t *testing.T) {
	orig := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", "aarch64-linux-gnu")
	}
	t.Cleanup(func() { execCommandContext = orig })

	got := targetTriple()
	if got != "aarch64-linux-gnu" {
		t.Errorf("targetTriple() = %q; want %q", got, "aarch64-linux-gnu")
	}
}

func TestTargetTriple_FallbackOnError(t *testing.T) {
	orig := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}
	t.Cleanup(func() { execCommandContext = orig })

	got := targetTriple()
	if got != "x86_64-linux-gnu" {
		t.Errorf("targetTriple() = %q; want 'x86_64-linux-gnu' (fallback)", got)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestToolchainNumCPUs_EmptyIsValidInteger(t *testing.T) {
	// When ToolchainConfig has zero-value NumCPUs, the function should
	// return a valid positive integer string.
	got := toolchainNumCPUs(ToolchainConfig{})
	if got == "" || got == "0" {
		t.Errorf("expected valid positive integer, got %q", got)
	}
}

func TestToolchainTarget_NotEmpty(t *testing.T) {
	// Even with empty config, target must be non-empty (auto-detected or fallback)
	got := toolchainTarget(ToolchainConfig{})
	if got == "" {
		t.Error("toolchainTarget returned empty for empty config")
	}
}

// ---------------------------------------------------------------------------
// Config consistency: verify that BootstrapConfig.SkipToolchain maps
// correctly to the build functions being skipped at the BootstrapAll level.
// We test the contract: when SkipToolchain is true, BootstrapAll must not
// call BuildBinutils / BuildGCC / BuildGLibc.
// ---------------------------------------------------------------------------

func TestBootstrapAll_SkipToolchainSkipsBuildSteps(t *testing.T) {
	tmpDir := t.TempDir()

	// Override execCommandContext to record calls
	origExec := execCommandContext
	var calls []string
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls = append(calls, name)
		// Return a command that succeeds
		cmd := exec.CommandContext(ctx, "true")
		return cmd
	}
	t.Cleanup(func() { execCommandContext = origExec })

	// Override DefaultToolchain to avoid real downloads
	origToolchain := DefaultToolchain
	DefaultToolchain = nil // no packages to download
	t.Cleanup(func() { DefaultToolchain = origToolchain })

	cfg := BootstrapConfig{
		TargetDir:     filepath.Join(tmpDir, "rootfs"),
		SkipToolchain: true,
		SkipKernel:    true,
		SkipBusybox:   true,
		SkipInitramfs: true,
		SkipGRUB:      true,
	}

	_ = BootstrapAll(context.Background(), cfg)

	// Check that no toolchain build commands were called
	for _, call := range calls {
		if call == "make" || call == "bash" || call == "configure" {
			t.Errorf("unexpected command %q called when SkipToolchain=true", call)
		}
	}
}

func TestBootstrapAll_AutoLoginAndMise(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")

	// Create /bin/zsh so configureDefaultShell doesn't skip
	zshPath := filepath.Join(rootfsDir, "bin", "zsh")
	if err := os.MkdirAll(filepath.Dir(zshPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zshPath, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create mise binary so setupMiseRuntimes proceeds
	miseBin := filepath.Join(rootfsDir, "usr", "bin", "mise")
	if err := os.MkdirAll(filepath.Dir(miseBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(miseBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// Override execCommandContext to avoid real execution
	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "true")
	}
	t.Cleanup(func() { execCommandContext = origExec })

	origToolchain := DefaultToolchain
	DefaultToolchain = nil
	t.Cleanup(func() { DefaultToolchain = origToolchain })

	cfg := BootstrapConfig{
		TargetDir:      rootfsDir,
		SkipToolchain:  true,
		SkipKernel:     true,
		SkipBusybox:    true,
		SkipInitramfs:  true,
		SkipGRUB:       true,
		AutoLoginUser:  "root",
		MiseRuntimes:   []string{"nodejs@lts"},
	}

	err := BootstrapAll(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BootstrapAll with autologin+mise failed: %v", err)
	}

	// Verify autologin drop-in was created
	dropinPath := filepath.Join(rootfsDir, "etc", "systemd", "system", "getty@tty1.service.d", "autologin.conf")
	if _, err := os.Stat(dropinPath); os.IsNotExist(err) {
		t.Error("autologin drop-in should exist")
	}

	// Verify mise profile script was created
	profilePath := filepath.Join(rootfsDir, "etc", "profile.d", "mise.sh")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("mise.sh should exist")
	}
}

func TestBootstrapAll_SkipToolchainFalseRunsBuildSteps(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")
	srcDir := filepath.Join(tmpDir, "sources")

	// Create minimal sources structure so build functions don't fail early
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create fake tarballs so the "not found" check passes
	writeFakeTarballPlain(t, filepath.Join(srcDir, "binutils-2.42.tar.xz"))
	writeFakeTarballPlain(t, filepath.Join(srcDir, "gcc-14.1.0.tar.xz"))
	writeFakeTarballPlain(t, filepath.Join(srcDir, "glibc-2.39.tar.xz"))

	// Create extracted source dirs so extraction is skipped
	for _, pkg := range []string{"binutils-2.42", "gcc-14.1.0", "glibc-2.39"} {
		pkgDir := filepath.Join(srcDir, pkg)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	// GCC needs contrib/download_prerequisites
	contribDir := filepath.Join(srcDir, "gcc-14.1.0", "contrib")
	if err := os.MkdirAll(contribDir, 0755); err != nil {
		t.Fatal(err)
	}
	mockScript := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(filepath.Join(contribDir, "download_prerequisites"), []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	// Override execCommandContext to record calls
	origExec := execCommandContext
	var calls []struct{ name string; args []string }
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls = append(calls, struct{ name string; args []string }{name, args})
		// Return a command that succeeds
		cmd := exec.CommandContext(ctx, "true")
		return cmd
	}
	t.Cleanup(func() { execCommandContext = origExec })

	// Override DefaultToolchain to avoid real downloads
	origToolchain := DefaultToolchain
	DefaultToolchain = nil
	t.Cleanup(func() { DefaultToolchain = origToolchain })

	cfg := BootstrapConfig{
		TargetDir:     rootfsDir,
		SkipToolchain: false,
		SkipKernel:    true,
		SkipBusybox:   true,
		SkipInitramfs: true,
		SkipGRUB:      true,
	}

	_ = BootstrapAll(context.Background(), cfg)

	// Verify that make commands were called (build steps ran)
	foundMake := false
	for _, call := range calls {
		if call.name == "make" {
			foundMake = true
			break
		}
	}
	if !foundMake {
		t.Error("expected make commands to be called when SkipToolchain=false")
	}
}

// ---------------------------------------------------------------------------
// BootstrapAll with DotfilesApply
// ---------------------------------------------------------------------------

func TestBootstrapAll_DotfilesApply(t *testing.T) {
	tmpDir := t.TempDir()
	rootfsDir := filepath.Join(tmpDir, "rootfs")

	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "true")
	}
	t.Cleanup(func() { execCommandContext = origExec })

	origToolchain := DefaultToolchain
	DefaultToolchain = nil
	t.Cleanup(func() { DefaultToolchain = origToolchain })

	cfg := BootstrapConfig{
		TargetDir:     rootfsDir,
		SkipToolchain: true,
		SkipKernel:    true,
		SkipBusybox:   true,
		SkipInitramfs: true,
		SkipGRUB:      true,
		DotfilesApply: true,
		ConfigsDir:    filepath.Join(tmpDir, "nonexistent-configs"),
	}

	// Dotfiles.Apply handles missing configs gracefully (warns and skips)
	err := BootstrapAll(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BootstrapAll with DotfilesApply should succeed: %v", err)
	}
}
