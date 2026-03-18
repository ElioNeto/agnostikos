package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ElioNeto/agnostikos/cmd/agnostic"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "Test Execute Function",
			expected: "Installation completed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tmpDir := t.TempDir()

			// Mock the os.Stdout to capture output
			oldStdout := os.Stdout
			os.Stdout = new(bytes.Buffer)
			defer func() { os.Stdout = oldStdout }()

			// Call the Execute function
			agnostic.Execute()

			// Check if the expected output is present in os.Stdout
			output := os.Stdout.(*bytes.Buffer).String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output not found: %s", tt.expected)
			}
		})
	}
}