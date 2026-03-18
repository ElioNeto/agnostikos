package agnostic

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestInstallCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flagBackend string
		flagIsolated bool
		wantErr     bool
	}{
		{"valid package", []string{"package1"}, "pacman", false, false},
		{"invalid backend", []string{"package1"}, "unknown", false, true},
		{"isolated install", []string{"package1"}, "pacman", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend = tt.flagBackend
			isolated = tt.flagIsolated

			cmd := &cobra.Command{}
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			err := installCmd.ExecuteContext(context.Background(), cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("installCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			output := buf.String()
			if !strings.Contains(output, "Installing 'package1' via") {
				t.Errorf("output does not contain expected installation message")
			}
			if isolated && !strings.Contains(output, "Running in isolated namespace...") {
				t.Errorf("output does not contain expected isolated namespace message")
			}
		})
	}
}

func TestRemoveCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flagBackend string
		wantErr     bool
	}{
		{"valid package", []string{"package1"}, "pacman", false},
		{"invalid backend", []string{"package1"}, "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend = tt.flagBackend

			cmd := &cobra.Command{}
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			err := removeCmd.ExecuteContext(context.Background(), cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			output := buf.String()
			if !strings.Contains(output, "Removing 'package1' via") {
				t.Errorf("output does not contain expected removal message")
			}
		})
	}
}

func TestUpdateCmd(t *testing.T) {
	tests := []struct {
		name        string
		flagBackend string
		wantErr     bool
	}{
		{"valid backend", "pacman", false},
		{"invalid backend", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend = tt.flagBackend

			cmd := &cobra.Command{}
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			err := updateCmd.ExecuteContext(context.Background(), cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			output := buf.String()
			if !strings.Contains(output, "Updating via") {
				t.Errorf("output does not contain expected update message")
			}
		})
	}
}

func TestSearchCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flagBackend string
		wantErr     bool
	}{
		{"valid query", []string{"query1"}, "pacman", false},
		{"invalid backend", []string{"query1"}, "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend = tt.flagBackend

			cmd := &cobra.Command{}
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			err := searchCmd.ExecuteContext(context.Background(), cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("searchCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			output := buf.String()
			if !strings.Contains(output, "Searching 'query1' in") {
				t.Errorf("output does not contain expected search message")
			}
		})
	}
}

type MockService struct {
	InstallFn  func(string) error
	RemoveFn   func(string) error
	UpdateFn   func() error
	SearchFn   func(string) ([]string, error)
	InstallErr error
	RemoveErr  error
	UpdateErr  error
	SearchErr  error
}

func (m *MockService) Install(packageName string) error {
	return m.InstallFn(packageName)
}

func (m *MockService) Remove(packageName string) error {
	return m.RemoveFn(packageName)
}

func (m *MockService) Update() error {
	return m.UpdateFn()
}

func (m *MockService) Search(query string) ([]string, error) {
	return m.SearchFn(query)
}

func TestInstallCmdWithMock(t *testing.T) {
	mock := &MockService{
		InstallFn: func(packageName string) error {
			if packageName != "package1" {
				return errors.New("unexpected package name")
			}
			return nil
		},
	}

	mgr := manager.AgnosticManager{Backends: map[string]manager.Service{"pacman": mock}}
	manager.NewAgnosticManager = func() *manager.AgnosticManager {
		return &mgr
	}

	tests := []struct {
		name        string
		args        []string
		flagBackend string
		flagIsolated bool
		wantErr     bool
	}{
		{"valid package", []string{"package1"}, "pacman", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend = tt.flagBackend
			isolated = tt.flagIsolated

			cmd := &cobra.Command{}
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			err := installCmd.ExecuteContext(context.Background(), cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("installCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			output := buf.String()
			if !strings.Contains(output, "Installing 'package1' via") {
				t.Errorf("output does not contain expected installation message")
			}
		})
	}
}