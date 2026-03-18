package agnostic

import (
	"errors"
	"testing"

	"github.com/ElioNeto/agnostikos/internal/manager"
)

type mockService struct {
	installFn   func(string) error
	removeFn    func(string) error
	updateFn    func() error
	searchFn    func(string) ([]string, error)
	isInstalled bool
}

func (m *mockService) Install(name string) error { return m.installFn(name) }
func (m *mockService) Remove(name string) error  { return m.removeFn(name) }
func (m *mockService) Update() error             { return m.updateFn() }
func (m *mockService) Search(query string) ([]string, error) {
	return m.searchFn(query)
}
func (m *mockService) IsInstalled(name string) bool { return m.isInstalled }

func TestInstallCmd(t *testing.T) {
	tests := []struct {
		name      string
		backend   string
		isolated  bool
		installFn func(string) error
		wantErr   bool
	}{
		{"Pacman Install", "pacman", false, nil, false},
		{"Nix Install", "nix", true, errors.New("fake install error"), true},
		{"Flatpak Install", "flatpak", false, errors.New("fake install error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &manager.AgnosticManager{Backends: map[string]manager.Backend{
				tt.backend: &mockService{installFn: tt.installFn},
			}}
			cmd := &cobra.Command{}
			cmd.Flags().StringVarP(&backend, "backend", "b", tt.backend, "")
			cmd.Flags().BoolVarP(&isolated, "isolated", "i", tt.isolated, "")
			err := installCmd.RunE(cmd, []string{"package"})
			if (err != nil) != tt.wantErr {
				t.Errorf("installCmd.RunE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveCmd(t *testing.T) {
	tests := []struct {
		name      string
		backend   string
		removeFn  func(string) error
		wantErr   bool
	}{
		{"Pacman Remove", "pacman", nil, false},
		{"Nix Remove", "nix", errors.New("fake remove error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &manager.AgnosticManager{Backends: map[string]manager.Backend{
				tt.backend: &mockService{removeFn: tt.removeFn},
			}}
			cmd := &cobra.Command{}
			cmd.Flags().StringVarP(&backend, "backend", "b", tt.backend, "")
			err := removeCmd.RunE(cmd, []string{"package"})
			if (err != nil) != tt.wantErr {
				t.Errorf("removeCmd.RunE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateCmd(t *testing.T) {
	tests := []struct {
		name      string
		backend   string
		updateFn  func() error
		wantErr   bool
	}{
		{"Pacman Update", "pacman", nil, false},
		{"Nix Update", "nix", errors.New("fake update error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &manager.AgnosticManager{Backends: map[string]manager.Backend{
				tt.backend: &mockService{updateFn: tt.updateFn},
			}}
			cmd := &cobra.Command{}
			cmd.Flags().StringVarP(&backend, "backend", "b", tt.backend, "")
			err := updateCmd.RunE(cmd, []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("updateCmd.RunE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSearchCmd(t *testing.T) {
	tests := []struct {
		name      string
		backend   string
		searchFn  func(string) ([]string, error)
		wantErr   bool
	}{
		{"Pacman Search", "pacman", nil, false},
		{"Nix Search", "nix", errors.New("fake search error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &manager.AgnosticManager{Backends: map[string]manager.Backend{
				tt.backend: &mockService{searchFn: tt.searchFn},
			}}
			cmd := &cobra.Command{}
			cmd.Flags().StringVarP(&backend, "backend", "b", tt.backend, "")
			err := searchCmd.RunE(cmd, []string{"query"})
			if (err != nil) != tt.wantErr {
				t.Errorf("searchCmd.RunE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}