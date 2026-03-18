package main

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecute(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
	}{
		{
			name: "Test Execute with default config flag",
			expected: `AgnosticOS CLI
Version: 0.1.0`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd.SetArgs([]string{"--config", ""})
			buf := &strings.Builder{}
			rootCmd.SetOut(buf)

			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			output := buf.String()
			if output != tc.expected {
				t.Errorf("Expected output to be:\n%s\nbut got:\n%s", tc.expected, output)
			}
		})
	}
}

func TestInitConfigFlag(t *testing.T) {
	testCases := []struct {
		name     string
		flag     string
		expected string
	}{
		{
			name:     "Test init with config flag",
			flag:     "config",
			expected: "$HOME/.agnostikos.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd.Flags().StringP(tc.flag, "", "", "config file (default is $HOME/.agnostikos.yaml)")
			val := rootCmd.Flag(tc.flag).Value.String()
			if val != tc.expected {
				t.Errorf("Expected flag value to be %s but got %s", tc.expected, val)
			}
		})
	}
}