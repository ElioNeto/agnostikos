//go:build helper

// This file is a helper for building the initramfs from the command line.
// It is excluded from normal test runs using the "helper" build tag.
// Usage: go test -tags=helper -run ^TestBuildInitramfsHelper$ ./internal/bootstrap/ -args <rootfsDir> <outputPath>

package bootstrap

import (
	"context"
	"os"
	"testing"
)

func TestBuildInitramfsHelper(t *testing.T) {
	if len(os.Args) < 5 { // -test.run, -test.args, rootfsDir, outputPath... actually os.Args includes test flags
		t.Fatal("Usage: go test -tags=helper -run ^TestBuildInitramfsHelper$ ./internal/bootstrap/ -- <rootfsDir> <outputPath>")
	}
	// Find the positional args after test flags
	var args []string
	for i, a := range os.Args {
		if a == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	if len(args) < 2 {
		t.Fatal("Usage: ... -- <rootfsDir> <outputPath>")
	}
	rootfsDir := args[0]
	outputPath := args[1]

	if err := BuildInitramfs(context.Background(), rootfsDir, outputPath); err != nil {
		t.Fatalf("BuildInitramfs failed: %v", err)
	}
}
