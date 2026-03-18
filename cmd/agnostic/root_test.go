package agnostic

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"help", []string{"--help"}, false},
		{"version", []string{"--version"}, false},
		{"invalid", []string{"invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("agnostic", tt.args...)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			err := cmd.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && out.String() == "" {
				t.Errorf("Expected output but got none")
			}
		})
	}
}