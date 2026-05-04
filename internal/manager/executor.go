package manager

import (
	"context"
	"os/exec"
)

type Executor interface {
	RunContext(ctx context.Context, name string, args ...string) ([]byte, error)
}

type RealExecutor struct{}

func (r *RealExecutor) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
