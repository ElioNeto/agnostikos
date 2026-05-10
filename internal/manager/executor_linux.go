//go:build linux

package manager

import (
	"context"
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/isolation"
)

// IsolatedExecutor wraps command execution with Linux namespace isolation
// (mount, PID, UTS, IPC). All commands run through the isolation package
// to prevent side effects on the host system.
//
// Requires CAP_SYS_ADMIN. Falls back to non-isolated execution only when
// the platform does not support namespaces (not applicable on Linux).
type IsolatedExecutor struct{}

// RunContext executes a command with full namespace isolation and returns
// the combined standard output and standard error.
func (e *IsolatedExecutor) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := isolation.RunIsolatedWithOutput(ctx, name, args)
	if err != nil {
		return out, fmt.Errorf("isolated: %w", err)
	}
	return out, nil
}
