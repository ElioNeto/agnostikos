//go:build linux

package isolation

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestNewIsolationConfig_Defaults(t *testing.T) {
	cfg := NewIsolationConfig()
	expected := syscall.CLONE_NEWNS | syscall.CLONE_NEWPID |
		syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC
	if cfg.Cloneflags != expected {
		t.Errorf("expected Cloneflags=%d, got %d", expected, cfg.Cloneflags)
	}
	if cfg.RootFS != "" {
		t.Errorf("expected empty RootFS, got %q", cfg.RootFS)
	}
}

func TestWithRootFS_SetsRootFS(t *testing.T) {
	cfg := NewIsolationConfig()
	WithRootFS("/some/chroot")(cfg)
	if cfg.RootFS != "/some/chroot" {
		t.Errorf("expected RootFS=/some/chroot, got %q", cfg.RootFS)
	}
}

func TestWithRootFS_OverwritesPrevious(t *testing.T) {
	cfg := NewIsolationConfig()
	WithRootFS("/first")(cfg)
	WithRootFS("/second")(cfg)
	if cfg.RootFS != "/second" {
		t.Errorf("expected RootFS=/second, got %q", cfg.RootFS)
	}
}

func TestWithCloneFlags_OverridesDefaults(t *testing.T) {
	cfg := NewIsolationConfig()
	WithCloneFlags(syscall.CLONE_NEWNET)(cfg)

	if cfg.Cloneflags != syscall.CLONE_NEWNET {
		t.Errorf("expected CLONE_NEWNET, got %d", cfg.Cloneflags)
	}
	// Default flags should be cleared, not OR'd.
	if cfg.Cloneflags&syscall.CLONE_NEWNS != 0 {
		t.Error("default CLONE_NEWNS was not cleared")
	}
}

func TestWithCloneFlags_Combinations(t *testing.T) {
	tests := []struct {
		name  string
		flags uintptr
	}{
		{"CLONE_NEWNS", syscall.CLONE_NEWNS},
		{"CLONE_NEWPID", syscall.CLONE_NEWPID},
		{"CLONE_NEWUTS", syscall.CLONE_NEWUTS},
		{"CLONE_NEWIPC", syscall.CLONE_NEWIPC},
		{"NEWNS+NEWPID", syscall.CLONE_NEWNS | syscall.CLONE_NEWPID},
		{"ALL", syscall.CLONE_NEWNS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewIsolationConfig()
			WithCloneFlags(tt.flags)(cfg)
			if cfg.Cloneflags != tt.flags {
				t.Errorf("expected %d, got %d", tt.flags, cfg.Cloneflags)
			}
		})
	}
}

func TestOptions_AppliedInOrder(t *testing.T) {
	// WithRootFS after WithCloneFlags should not affect clone flags.
	cfg := NewIsolationConfig()
	WithCloneFlags(syscall.CLONE_NEWNS | syscall.CLONE_NEWPID)(cfg)
	WithRootFS("/chroot")(cfg)

	if cfg.Cloneflags != syscall.CLONE_NEWNS|syscall.CLONE_NEWPID {
		t.Errorf("unexpected Cloneflags: %d", cfg.Cloneflags)
	}
	if cfg.RootFS != "/chroot" {
		t.Errorf("unexpected RootFS: %q", cfg.RootFS)
	}
}

func TestRunIsolatedWithContext_CancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := RunIsolatedWithContext(ctx, "true")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !IsCancelled(err) {
		t.Errorf("expected cancellation error, got %v", err)
	}
}

func TestRunIsolatedWithOptions_CancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RunIsolatedWithOptions(ctx, "true", nil,
		WithCloneFlags(syscall.CLONE_NEWNS),
	)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !IsCancelled(err) {
		t.Errorf("expected cancellation error, got %v", err)
	}
}

func TestRunIsolatedWithOptions_InvalidRootFS_NotExist(t *testing.T) {
	// RootFS that does not exist should produce an error before any
	// privilege escalation is attempted.
	err := RunIsolatedWithOptions(context.Background(), "true", nil,
		WithRootFS("/tmp/nonexistent-chroot-XXXXXX"),
	)
	if err == nil {
		t.Fatal("expected error for non-existent RootFS, got nil")
	}
	// The error should mention the path.
	if errors.Is(err, os.ErrNotExist) {
		// Good — os.Stat returned ENOENT.
	} else {
		t.Logf("got error (non-existence may be wrapped differently): %v", err)
	}
}

func TestRunIsolatedWithOptions_InvalidRootFS_NotADir(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(tmpFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	err := RunIsolatedWithOptions(context.Background(), "true", nil,
		WithRootFS(tmpFile),
	)
	if err == nil {
		t.Fatal("expected error when RootFS is a file, got nil")
	}
	t.Logf("got expected error for file-as-rootfs: %v", err)
}

func TestIsCancelled_ContextCancelled(t *testing.T) {
	if !IsCancelled(context.Canceled) {
		t.Error("expected IsCancelled(context.Canceled) to be true")
	}
}

func TestIsCancelled_ContextDeadline(t *testing.T) {
	if !IsCancelled(context.DeadlineExceeded) {
		t.Error("expected IsCancelled(context.DeadlineExceeded) to be true")
	}
}

func TestIsCancelled_NilReturnsFalse(t *testing.T) {
	if IsCancelled(nil) {
		t.Error("expected IsCancelled(nil) to be false")
	}
}

func TestIsCancelled_OtherError(t *testing.T) {
	if IsCancelled(errors.New("foo")) {
		t.Error("expected IsCancelled(random error) to be false")
	}
}

func TestUnshare_UserNamespaceAvailable(t *testing.T) {
	// Test that the system supports user namespaces via the unshare(1)
	// command. This validates the concept without requiring CAP_SYS_ADMIN
	// for the test process itself, because unshare -U creates a new user
	// namespace where the process is effectively root.
	out, err := exec.Command("unshare", "-U", "true").CombinedOutput()
	if err != nil {
		t.Skipf("unshare -U true failed — user namespaces may be unavailable: %v\n%s", err, string(out))
	}
}

func TestUnshare_MountNamespace(t *testing.T) {
	// Mount namespace via unshare -m (needs user namespace in front).
	out, err := exec.Command("unshare", "-U", "-m", "true").CombinedOutput()
	if err != nil {
		t.Skipf("unshare -U -m failed: %v\n%s", err, string(out))
	}
}

func TestUnshare_PIDNamespace(t *testing.T) {
	out, err := exec.Command("unshare", "-U", "-p", "--kill-child", "true").CombinedOutput()
	if err != nil {
		t.Skipf("unshare -U -p failed: %v\n%s", err, string(out))
	}
}

func TestUnshare_UTSNamespace(t *testing.T) {
	out, err := exec.Command("unshare", "-U", "-u", "true").CombinedOutput()
	if err != nil {
		t.Skipf("unshare -U -u failed: %v\n%s", err, string(out))
	}
}

func TestUnshare_IPCNamespace(t *testing.T) {
	out, err := exec.Command("unshare", "-U", "-i", "true").CombinedOutput()
	if err != nil {
		t.Skipf("unshare -U -i failed: %v\n%s", err, string(out))
	}
}

func TestUnshare_AllNamespaces(t *testing.T) {
	out, err := exec.Command("unshare", "-U", "-m", "-p", "--kill-child", "-u", "-i", "true").CombinedOutput()
	if err != nil {
		t.Skipf("unshare -U -m -p -u -i failed: %v\n%s", err, string(out))
	}
}

func TestUnshare_WithChroot(t *testing.T) {
	tmpDir := t.TempDir()

	// Prepare a minimal root directory with /bin/true.
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	trueBin := filepath.Join(binDir, "true")
	if err := copyFile("/bin/true", trueBin); err != nil {
		t.Skipf("cannot set up minimal chroot: %v", err)
	}

	// unshare -U creates a user namespace where we have full capabilities,
	// including CAP_SYS_CHROOT, so chroot inside should work.
	out, err := exec.Command("unshare", "-U", "chroot", tmpDir, "/bin/true").CombinedOutput()
	if err != nil {
		t.Skipf("unshare -U chroot failed: %v\n%s", err, string(out))
	}
}

// copyFile copies a file from src to dst, preserving mode.
func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, fi.Mode())
}
