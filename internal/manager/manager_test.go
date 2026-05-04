package manager

import (
	"errors"
	"testing"
)

// MockBackend implementa PackageService para testes
type MockBackend struct {
	InstallErr error
	RemoveErr  error
	UpdateErr  error
	SearchRes  []string
	SearchErr  error
}

func (m *MockBackend) Install(pkgName string) error { return m.InstallErr }
func (m *MockBackend) Remove(pkgName string) error  { return m.RemoveErr }
func (m *MockBackend) Update() error                { return m.UpdateErr }
func (m *MockBackend) Search(query string) ([]string, error) {
	return m.SearchRes, m.SearchErr
}

func TestMockBackend_Install_Success(t *testing.T) {
	svc := &MockBackend{}
	if err := svc.Install("firefox"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestMockBackend_Install_Error(t *testing.T) {
	svc := &MockBackend{InstallErr: errors.New("install failed")}
	if err := svc.Install("firefox"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMockBackend_Remove_Success(t *testing.T) {
	svc := &MockBackend{}
	if err := svc.Remove("firefox"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestMockBackend_Update_Success(t *testing.T) {
	svc := &MockBackend{}
	if err := svc.Update(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestMockBackend_Search_Success(t *testing.T) {
	svc := &MockBackend{SearchRes: []string{"firefox", "firefox-esr"}}
	results, err := svc.Search("firefox")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestMockBackend_Search_Error(t *testing.T) {
	svc := &MockBackend{SearchErr: errors.New("search failed")}
	_, err := svc.Search("firefox")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewAgnosticManager_HasAllBackends(t *testing.T) {
	mgr := NewAgnosticManager()
	for _, name := range []string{"pacman", "nix", "flatpak"} {
		if _, ok := mgr.Backends[name]; !ok {
			t.Errorf("expected backend '%s' to be registered", name)
		}
	}
}

func TestPacmanBackend_EmptyPackageName(t *testing.T) {
	p := &PacmanBackend{}
	if err := p.Install(""); err == nil {
		t.Error("expected error for empty package name")
	}
	if err := p.Remove(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestNixBackend_EmptyPackageName(t *testing.T) {
	n := &NixBackend{}
	if err := n.Install(""); err == nil {
		t.Error("expected error for empty package name")
	}
	if err := n.Remove(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestFlatpakBackend_EmptyPackageName(t *testing.T) {
	f := &FlatpakBackend{}
	if err := f.Install(""); err == nil {
		t.Error("expected error for empty package name")
	}
	if err := f.Remove(""); err == nil {
		t.Error("expected error for empty package name")
	}
}
