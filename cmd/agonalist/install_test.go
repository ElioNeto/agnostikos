package agnostic

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestInstallCmd(t *testing.T) {
	tests := []struct {
		name     string
		wantErr  bool
	}{
		{"no_error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", "cmd/agonalist/install.go", "install")
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}