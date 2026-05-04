//go:build linux

package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// RunIsolated executa o comando dado em um namespace de mount isolado (CLONE_NEWNS).
// Requer que o processo seja root ou tenha CAP_SYS_ADMIN.
func RunIsolated(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("isolated run failed: %w", err)
	}
	return nil
}
