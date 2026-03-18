package agnostic

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

// MockAgnosticManager for testing purposes
type MockAgnosticManager struct{}

func (m *MockAgnosticManager) Backends() map[string]Service {
	return map[string]Service{
		"pacman": &MockPacman{},
	}
}

// MockPacman for testing purposes
type MockPacman struct{}

func (p *MockPacman) Install(packageName string) error {
	if packageName == "error" {
		return errors.New("mock install error")
	}
	return nil
}

func (p *MockPacman) Remove(packageName string) error {
	if packageName == "error" {
		return errors.New("mock remove error")
	}
	return nil
}

func (p *MockPacman) Update() error {
	return nil
}

func (p *MockPacman) Search(query string) ([]string, error) {
	if query == "error" {
		return nil, errors.New("mock search error")
	}
	return []string{"result1", "result2"}, nil
}

var backend = "pacman"
var isolated = false

// TestInstall tests the install command with different scenarios.
func TestInstall(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
	}{
		{"valid package", []string{"package1"}, false},
		{"invalid backend", []string{"package2"}, true},
		{"install error", []string{"error"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
			err := installCmd.RunE(installCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRemove tests the remove command with different scenarios.
func TestRemove(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
	}{
		{"valid package", []string{"package1"}, false},
		{"invalid backend", []string{"package2"}, true},
		{"remove error", []string{"error"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			removeCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
			err := removeCmd.RunE(removeCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdate tests the update command with different scenarios.
func TestUpdate(t *testing.T) {
	tests := []struct {
		name     string
		wantErr  bool
	}{
		{"valid backend", false},
		{"invalid backend", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
			err := updateCmd.RunE(updateCmd, []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSearch tests the search command with different scenarios.
func TestSearch(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
	}{
		{"valid query", []string{"query1"}, false},
		{"invalid backend", []string{"query2"}, true},
		{"search error", []string{"error"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchCmd.Flags().StringVarP(&backend, "backend", "b", "pacman", "Backend to use (pacman, nix, flatpak)")
			err := searchCmd.RunE(searchCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}