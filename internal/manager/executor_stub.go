//go:build !linux

package manager

import (
	"context"
	"os/exec"
)

// IsolatedExecutor provides a no-op fallback on non-Linux platforms where
// namespace isolation is not available. Commands are executed without
// isolation.
type IsolatedExecutor struct{}

// RunContext executes a command without namespace isolation (non-Linux
// fallback) and returns the combined output.
func (e *IsolatedExecutor) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
