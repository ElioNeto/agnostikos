package agnostic

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "Test default version output",
			expected: `AgnosticOS Hybrid Package Manager version 0.1.0 (commit: dev)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cmd := exec.Command("go", "run", ".", "--version")
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				t.Errorf("cmd.Run() error = %v", err)
				return
			}
			actual := out.String()
			if actual != tt.expected {
				t.Errorf("Expected output:\n%s\nGot:\n%s", tt.expected, actual)
			}
		})
	}
}