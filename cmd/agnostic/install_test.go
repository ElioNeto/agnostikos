```go
package agnostic

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/ElioNeto/agnostikos/internal/manager"
)

func TestInstallCmd(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		isolated bool
		args     []string
		wantErr  bool
	}{
		{"Valid backend and package", "pacman", false, []string{"package1"}, false},
		{"Invalid backend", "unknown", false, []string{"package1"}, true},
		{"Isolated mode", "pacman", true, []string{"package1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			cmd := installCmd
			cmd.Flags().Set("backend", tt.backend)
			cmd.Flags().Set("isolated", fmt.Sprintf("%t", tt.isolated))
			err := cmd.RunE(cmd, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("RunE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			expectedOutput := fmt.Sprintf("📦 Installing '%s' via %s...\n", tt.args[0], tt.backend)
			if tt.isolated {
				expectedOutput += "🔒 Running in isolated namespace...\n"
			}
			expectedOutput += "✅ 'package1' installed successfully\n"

			if out.String() != expectedOutput {
				t.Errorf("RunE() output = %v, want %v", out.String(), expectedOutput)
			}
		})
	}
}

func TestRemoveCmd(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		args     []string
		wantErr  bool
	}{
		{"Valid backend and package", "pacman", []string{"package1"}, false},
		{"Invalid backend", "unknown", []string{"package1"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			cmd := removeCmd
			cmd.Flags().Set("backend", tt.backend)
			err := cmd.RunE(cmd, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("RunE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			expectedOutput := fmt.Sprintf("🗑️  Removing '%s' via %s...\n", tt.args[0], tt.backend)
			expectedOutput += "✅ 'package1' removed\n"

			if out.String() != expectedOutput {
				t.Errorf("RunE() output = %v, want %v", out.String(), expectedOutput)
			}
		})
	}
}

func TestUpdateCmd(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		wantErr  bool
	}{
		{"Valid backend", "pacman", false},
		{"Invalid backend", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			cmd := updateCmd
			cmd.Flags().Set("backend", tt.backend)
			err := cmd.RunE(cmd, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("RunE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			expectedOutput := fmt.Sprintf("🔄 Updating via %s...\n", tt.backend)
			expectedOutput += "✅ Update complete\n"

			if out.String() != expectedOutput {
				t.Errorf("RunE() output = %v, want %v", out.String(), expectedOutput)
			}
		})
	}
}

func TestSearchCmd(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		args     []string
		wantErr  bool
	}{
		{"Valid backend and query", "pacman", []string{"query1"}, false},
		{"Invalid backend", "unknown", []string{"query1"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			cmd := searchCmd
			cmd.Flags().Set("backend", tt.backend)
			err := cmd.RunE(cmd, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("RunE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			expectedOutput := fmt.Sprintf("🔍 Searching '%s' in %s...\n", tt.args[0], tt.backend)

			if out.String() != expectedOutput {
				t.Errorf("RunE() output = %v, want %v", out.String(), expectedOutput)
			}
		})
	}
}

func TestManagerBackend(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		wantErr  bool
	}{
		{"Valid backend", "pacman", false},
		{"Invalid backend", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := manager.NewAgnosticManager()
			svc, ok := mgr.Backends[tt.backend]

			if (ok != !tt.wantErr) || (svc == nil && !tt.wantErr) {
				t.Errorf("Backend lookup = %v, want %v", svc, tt.wantErr)
			}
		})
	}
}

func TestManagerInstall(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		args     []string
		wantErr  bool
	}{
		{"Valid backend and package", "pacman", []string{"package1"}, false},
		{"Invalid backend", "unknown", []string{"package1"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := manager.NewAgnosticManager()
			svc, ok := mgr.Backends[tt.backend]

			if !ok {
				t.Errorf("Backend lookup failed")
				return
			}

			err := svc.Install(tt.args[0])

			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerRemove(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		args     []string
		wantErr  bool
	}{
		{"Valid backend and package", "pacman", []string{"package1"}, false},
		{"Invalid backend", "unknown", []string{"package1"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := manager.NewAgnosticManager()
			svc, ok := mgr.Backends[tt.backend]

			if !ok {
				t.Errorf("Backend lookup failed")
				return
			}

			err := svc.Remove(tt.args[0])

			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerUpdate(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		wantErr  bool
	}{
		{"Valid backend", "pacman", false},
		{"Invalid backend", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := manager.NewAgnosticManager()
			svc, ok := mgr.Backends[tt.backend]

			if !ok {
				t.Errorf("Backend lookup failed")
				return
			}

			err := svc.Update()

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerSearch(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		args     []string
		wantErr  bool
	}{
		{"Valid backend and query", "pacman", []string{"query1"}, false},
		{"Invalid backend", "unknown", []string{"query1"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := manager.NewAgnosticManager()
			svc, ok := mgr.Backends[tt.backend]

			if !ok {
				t.Errorf("Backend lookup failed")