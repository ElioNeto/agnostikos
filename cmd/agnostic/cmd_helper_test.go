package agnostic

import (
	"os/exec"
	"testing"
)

// skipIfNoBackend skips the test if the given backend binary is not
// available in PATH. This is needed because backends are now registered
// conditionally based on binary presence.
func skipIfNoBackend(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("skipping test: backend '%s' not available in PATH", name)
	}
}
