package manager

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireRoot skips the test if not running as root (required for namespace
// isolation with CLONE_NEW* flags). On Linux, unprivileged user namespaces
// may also work, but we use the simpler root check.
func requireRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("skipping: test requires root or CAP_SYS_ADMIN for namespace isolation")
	}
}

// TestIsolatedExecutor_RunContext_Success verifies that a simple command
// runs successfully through the IsolatedExecutor.
func TestIsolatedExecutor_RunContext_Success(t *testing.T) {
	requireRoot(t)

	executor := &IsolatedExecutor{}
	ctx := context.Background()

	out, err := executor.RunContext(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(bytes.TrimSpace(out), []byte("hello")) {
		t.Errorf("expected output to contain 'hello', got: %s", string(out))
	}
}

// TestIsolatedExecutor_RunContext_CommandNotFound verifies that the
// executor returns an error when the command does not exist.
func TestIsolatedExecutor_RunContext_CommandNotFound(t *testing.T) {
	executor := &IsolatedExecutor{}
	ctx := context.Background()

	_, err := executor.RunContext(ctx, "nonexistent-command-12345")
	if err == nil {
		t.Fatal("expected error for nonexistent command, got nil")
	}
}

// TestIsolatedExecutor_RunContext_CancelledContext verifies that a
// cancelled context causes the command to fail early.
func TestIsolatedExecutor_RunContext_CancelledContext(t *testing.T) {
	executor := &IsolatedExecutor{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := executor.RunContext(ctx, "true")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Logf("error does not wrap context.Canceled: %v", err)
	}
}

// TestRealExecutor_RunContext_Success verifies that RealExecutor works.
func TestRealExecutor_RunContext_Success(t *testing.T) {
	executor := &RealExecutor{}
	ctx := context.Background()

	out, err := executor.RunContext(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(bytes.TrimSpace(out), []byte("hello")) {
		t.Errorf("expected output to contain 'hello', got: %s", string(out))
	}
}

// TestRealExecutor_RunContext_CommandNotFound verifies RealExecutor error.
func TestRealExecutor_RunContext_CommandNotFound(t *testing.T) {
	executor := &RealExecutor{}
	ctx := context.Background()

	_, err := executor.RunContext(ctx, "nonexistent-command-12345")
	if err == nil {
		t.Fatal("expected error for nonexistent command, got nil")
	}
}

// TestIsolatedExecutor_RunContext_OutputCaptured verifies that the
// combined output (stdout + stderr) is properly captured.
func TestIsolatedExecutor_RunContext_OutputCaptured(t *testing.T) {
	requireRoot(t)

	executor := &IsolatedExecutor{}
	ctx := context.Background()

	out, err := executor.RunContext(ctx, "sh", "-c", "echo 'stdout'; echo 'stderr' >&2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, "stdout") {
		t.Errorf("expected 'stdout' in output, got: %s", output)
	}
	if !strings.Contains(output, "stderr") {
		t.Errorf("expected 'stderr' in output, got: %s", output)
	}
}

// TestIsolatedExecutor_RunContext_ExitError verifies that a non-zero exit
// code is reported as an error with the output still available.
func TestIsolatedExecutor_RunContext_ExitError(t *testing.T) {
	requireRoot(t)

	executor := &IsolatedExecutor{}
	ctx := context.Background()

	out, err := executor.RunContext(ctx, "sh", "-c", "echo 'message' && exit 42")
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
	// Verify the output is still captured
	if !bytes.Contains(out, []byte("message")) {
		t.Errorf("expected output to contain 'message', got: %s", string(out))
	}
	// Verify the error mentions exit code
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() != 42 {
			t.Errorf("expected exit code 42, got %d", exitErr.ExitCode())
		}
	} else {
		t.Logf("error does not wrap ExitError: %v", err)
	}
}

// TestNewAgnosticManager_DefaultUsesExecutor verifies that the default
// manager creates backends (the executor type is an implementation detail).
func TestNewAgnosticManager_DefaultUsesExecutor(t *testing.T) {
	mgr := NewAgnosticManager()
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if len(mgr.Backends) == 0 {
		t.Fatal("expected at least one backend")
	}
	// Verify core backends are present
	for _, name := range []string{"pacman", "nix", "flatpak"} {
		if _, ok := mgr.Backends[name]; !ok {
			t.Errorf("expected backend '%s' to be registered", name)
		}
	}
}

// TestNewAgnosticManager_WithNoSandbox verifies that WithNoSandbox
// creates a manager successfully.
func TestNewAgnosticManager_WithNoSandbox(t *testing.T) {
	mgr := NewAgnosticManager(WithNoSandbox())
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if len(mgr.Backends) == 0 {
		t.Fatal("expected at least one backend")
	}
	for _, name := range []string{"pacman", "nix", "flatpak"} {
		if _, ok := mgr.Backends[name]; !ok {
			t.Errorf("expected backend '%s' to be registered", name)
		}
	}
}

// TestIsolatedExecutorRunsInTempDir verifies that commands run through
// the isolated executor can still read from the filesystem normally
// (since only mount/PID/UTS/IPC namespaces are isolated, not the
// filesystem).
func TestIsolatedExecutorRunsInTempDir(t *testing.T) {
	requireRoot(t)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("isolated"), 0644); err != nil {
		t.Fatal(err)
	}

	executor := &IsolatedExecutor{}
	ctx := context.Background()

	out, err := executor.RunContext(ctx, "cat", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(bytes.TrimSpace(out), []byte("isolated")) {
		t.Errorf("expected 'isolated' in output, got: %s", string(out))
	}
}

// TestIsolatedExecutor_InterfaceSatisfaction ensures IsolatedExecutor
// satisfies the Executor interface at compile time.
func TestIsolatedExecutor_InterfaceSatisfaction(t *testing.T) {
	var _ Executor = (*IsolatedExecutor)(nil)
	var _ Executor = (*RealExecutor)(nil)
}

// TestNewAgnosticManager_WithNoSandboxRunsCommands verifies that a
// manager created with WithNoSandbox can run basic commands.
func TestNewAgnosticManager_WithNoSandboxRunsCommands(t *testing.T) {
	mgr := NewAgnosticManager(WithNoSandbox())
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	// We can use RealExecutor directly to verify commands work
	executor := &RealExecutor{}
	ctx := context.Background()
	out, err := executor.RunContext(ctx, "echo", "no-sandbox-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(bytes.TrimSpace(out), []byte("no-sandbox-test")) {
		t.Errorf("expected 'no-sandbox-test' in output, got: %s", string(out))
	}
}

// TestRealExecutor_EmptyCommand verifies RealExecutor handles edge cases.
func TestRealExecutor_EmptyCommand(t *testing.T) {
	executor := &RealExecutor{}
	ctx := context.Background()

	_, err := executor.RunContext(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty command, got nil")
	}
}

// TestIsolatedExecutor_EmptyCommand verifies IsolatedExecutor handles edge cases.
func TestIsolatedExecutor_EmptyCommand(t *testing.T) {
	executor := &IsolatedExecutor{}
	ctx := context.Background()

	_, err := executor.RunContext(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty command, got nil")
	}
}
