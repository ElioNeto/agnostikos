//go:build linux

// Package isolation provides Linux namespace isolation for running commands.
//
// It supports creating child processes in isolated Linux namespaces
// (mount, PID, UTS, IPC) with optional chroot via the functional options
// pattern.
//
// # Capability requirements
//
//   - Namespace creation (CLONE_NEWNS, CLONE_NEWPID, CLONE_NEWUTS, CLONE_NEWIPC)
//     requires CAP_SYS_ADMIN or that the caller is root.
//   - Chroot (WithRootFS) requires CAP_SYS_CHROOT.
//   - Tests use the unshare(1) command to demonstrate namespace behaviour
//     without requiring root privileges.
//
// # Portability
//
// This package is Linux-only. The build tag "//go:build linux" prevents
// compilation on other platforms. Tests may need the "linux" tag to run.
package isolation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// IsolationConfig holds the configuration for Linux namespace isolation.
//
// Create a default configuration with NewIsolationConfig() and apply
// overrides via IsolationOption functions.
type IsolationConfig struct {
	// Cloneflags are the Linux namespace flags passed to clone(2).
	// Default: CLONE_NEWNS | CLONE_NEWPID | CLONE_NEWUTS | CLONE_NEWIPC.
	Cloneflags uintptr

	// RootFS is an optional chroot target directory. When set, the
	// child process is chrooted into this directory before executing
	// the command. Requires CAP_SYS_CHROOT.
	RootFS string
}

// IsolationOption is a functional option for configuring IsolationConfig.
type IsolationOption func(*IsolationConfig)

// NewIsolationConfig returns an IsolationConfig with all common namespace
// flags enabled by default:
//
//   - CLONE_NEWNS  – mount namespace
//   - CLONE_NEWPID – PID namespace
//   - CLONE_NEWUTS – hostname/domainname namespace
//   - CLONE_NEWIPC – System V IPC namespace
func NewIsolationConfig() *IsolationConfig {
	return &IsolationConfig{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC,
	}
}

// WithRootFS returns an IsolationOption that sets the chroot directory.
// When configured, the command runs inside a chroot jail in combination
// with namespace isolation.
//
// The rootfs directory must exist and be a directory at the time the
// command is started. Requires CAP_SYS_CHROOT.
func WithRootFS(rootfs string) IsolationOption {
	return func(cfg *IsolationConfig) {
		cfg.RootFS = rootfs
	}
}

// WithCloneFlags returns an IsolationOption that overrides the default
// clone flags with the given flags. Use this to select exactly which
// namespaces the child process should be created in.
//
// Example:
//
//	WithCloneFlags(syscall.CLONE_NEWNS | syscall.CLONE_NEWPID)
func WithCloneFlags(flags uintptr) IsolationOption {
	return func(cfg *IsolationConfig) {
		cfg.Cloneflags = flags
	}
}

// RunIsolated runs the given command in an isolated mount namespace
// (CLONE_NEWNS).
//
// It is equivalent to:
//
//	RunIsolatedWithOptions(ctx, name, args,
//	    WithCloneFlags(syscall.CLONE_NEWNS),
//	)
//
// Requires CAP_SYS_ADMIN. For a context-aware variant see
// RunIsolatedWithContext.
func RunIsolated(name string, args ...string) error {
	return RunIsolatedWithContext(context.Background(), name, args...)
}

// RunIsolatedWithContext is like RunIsolated but accepts a cancellable
// context. The command runs inside a new mount namespace (CLONE_NEWNS).
//
// If the context is cancelled before or during execution the command
// is killed and the returned error wraps context.Canceled or
// context.DeadlineExceeded.
func RunIsolatedWithContext(ctx context.Context, name string, args ...string) error {
	return RunIsolatedWithOptions(ctx, name, args,
		WithCloneFlags(syscall.CLONE_NEWNS),
	)
}

// RunIsolatedWithOptions runs a command with full control over namespace
// isolation and optional chroot. Use the functional options to configure
// the isolation behaviour.
//
// Example — isolated command with PID + mount namespaces and a chroot:
//
//	err := RunIsolatedWithOptions(ctx, "/bin/sh", []string{"-c", "echo hello"},
//	    WithCloneFlags(syscall.CLONE_NEWNS|syscall.CLONE_NEWPID),
//	    WithRootFS("/mnt/chroot"),
//	)
//
// Requires CAP_SYS_ADMIN (and CAP_SYS_CHROOT if WithRootFS is used).
func RunIsolatedWithOptions(ctx context.Context, name string, args []string, options ...IsolationOption) error {
	cfg := NewIsolationConfig()
	for _, opt := range options {
		opt(cfg)
	}
	return runIsolated(ctx, cfg, name, args...)
}

// runIsolated is the internal implementation shared by public functions.
func runIsolated(ctx context.Context, cfg *IsolationConfig, name string, args ...string) error {
	// Fast-fail on cancelled context before any syscall.
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("isolated run cancelled: %w", err)
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: cfg.Cloneflags,
	}

	if cfg.RootFS != "" {
		fi, err := os.Stat(cfg.RootFS)
		if err != nil {
			return fmt.Errorf("chroot target %q: %w", cfg.RootFS, err)
		}
		if !fi.IsDir() {
			return fmt.Errorf("chroot target %q is not a directory", cfg.RootFS)
		}
		cmd.SysProcAttr.Chroot = cfg.RootFS
	}

	if err := cmd.Run(); err != nil {
		// Prefer the context error when the context was cancelled.
		if ctx.Err() != nil {
			return fmt.Errorf("isolated run cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("isolated run failed: %w", err)
	}
	return nil
}

// IsCancelled reports whether err is or wraps a context error
// (context.Canceled or context.DeadlineExceeded) typically returned
// by RunIsolatedWithContext or RunIsolatedWithOptions when the
// context is cancelled.
func IsCancelled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
