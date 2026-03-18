package main

import (
	"os"
	"testing"

	"github.com/ElioNeto/agnostikos/cmd/agnostic"
)

func TestExecute(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Mock the os.Stdout to capture output
	oldStdout := os.Stdout
	os.Stdout = new(bytes.Buffer)
	defer func() { os.Stdout = oldStdout }()

	// Call the Execute function
	agnostic.Execute()

	// Check if the expected output is present in os.Stdout
	expectedOutput := "Installation completed successfully"
	output := os.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected output not found: %s", expectedOutput)
	}
}