package manager

import (
	"context"
	"os/exec"
)

// Executor abstrai a execução de comandos externos
type Executor interface {
	RunContext(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RealExecutor chama exec.CommandContext de verdade
type RealExecutor struct{}

func (r *RealExecutor) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
