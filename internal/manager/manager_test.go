package manager

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

// MockExecutor substitui RealExecutor nos testes
type MockExecutor struct {
	Output []byte
	Err    error
}

func (m *MockExecutor) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.Output, m.Err
}

// --- PacmanBackend ---

func TestPacmanBackend_Install_EmptyName(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{}}
	if err := p.Install(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestPacmanBackend_Install_Success(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{Output: []byte("installed")}}
	if err := p.Install("firefox"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestPacmanBackend_Install_ExecError(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{Err: errors.New("pacman: not found")}}
	if err := p.Install("firefox"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestPacmanBackend_Remove_EmptyName(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{}}
	if err := p.Remove(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestPacmanBackend_Remove_Success(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{}}
	if err := p.Remove("firefox"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestPacmanBackend_Remove_ExecError(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{Err: errors.New("pacman: not found")}}
	if err := p.Remove("firefox"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestPacmanBackend_Update_EmptyName(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{}}
	if err := p.Update(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestPacmanBackend_Update_Success(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{}}
	if err := p.Update("firefox"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestPacmanBackend_Update_ExecError(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{Err: errors.New("update failed")}}
	if err := p.Update("firefox"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestPacmanBackend_UpdateAll_Success(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{}}
	if err := p.UpdateAll(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestPacmanBackend_UpdateAll_ExecError(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{Err: errors.New("update failed")}}
	if err := p.UpdateAll(); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestPacmanBackend_List_Success(t *testing.T) {
	output := "firefox 124.0-1\ngit 2.44.0\nlinux 6.6.0"
	p := &PacmanBackend{exec: &MockExecutor{Output: []byte(output)}}
	results, err := p.List()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestPacmanBackend_List_ExecError(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{Err: errors.New("list failed")}}
	if _, err := p.List(); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestPacmanBackend_Search_EmptyQuery(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{}}
	if _, err := p.Search(""); err == nil {
		t.Error("expected error for empty query")
	}
}

func TestPacmanBackend_Search_Success(t *testing.T) {
	output := "extra/firefox 124.0-1\n    Fast web browser\nextra/firefox-esr 115.0-1\n    Extended support release"
	p := &PacmanBackend{exec: &MockExecutor{Output: []byte(output)}}
	results, err := p.Search("firefox")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (package lines only), got %d", len(results))
	}
}

func TestPacmanBackend_Search_ExecError(t *testing.T) {
	p := &PacmanBackend{exec: &MockExecutor{Err: errors.New("search failed")}}
	if _, err := p.Search("firefox"); err == nil {
		t.Error("expected error, got nil")
	}
}

// --- NixBackend ---

func TestNixBackend_Install_EmptyName(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	if err := n.Install(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestNixBackend_Install_AddsPrefixAutomatically(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	// sem # no nome → deve adicionar nixpkgs# internamente sem erro
	if err := n.Install("neovim"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestNixBackend_Install_WithPrefix(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	if err := n.Install("nixpkgs#neovim"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestNixBackend_Install_ExecError(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{Err: errors.New("nix: not found")}}
	if err := n.Install("neovim"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNixBackend_Remove_EmptyName(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	if err := n.Remove(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestNixBackend_Remove_Success(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	if err := n.Remove("neovim"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestNixBackend_Remove_ExecError(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{Err: errors.New("remove failed")}}
	if err := n.Remove("neovim"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNixBackend_Update_EmptyName(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	if err := n.Update(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestNixBackend_Update_Success(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	if err := n.Update("neovim"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestNixBackend_Update_ExecError(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{Err: errors.New("upgrade failed")}}
	if err := n.Update("neovim"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNixBackend_UpdateAll_Success(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	if err := n.UpdateAll(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestNixBackend_UpdateAll_ExecError(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{Err: errors.New("upgrade failed")}}
	if err := n.UpdateAll(); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNixBackend_List_Success(t *testing.T) {
	output := `[
  {
    "active": true,
    "name": "nixpkgs.firefox",
    "version": "124.0"
  }
]`
	n := &NixBackend{exec: &MockExecutor{Output: []byte(output)}}
	results, err := n.List()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected non-empty results")
	}
}

func TestNixBackend_List_ExecError(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{Err: errors.New("list failed")}}
	if _, err := n.List(); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNixBackend_Search_EmptyQuery(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{}}
	if _, err := n.Search(""); err == nil {
		t.Error("expected error for empty query")
	}
}

func TestNixBackend_Search_Success(t *testing.T) {
	output := "* legacyPackages.x86_64-linux.neovim (0.9.5)\n  Vim text editor fork focused on extensibility"
	n := &NixBackend{exec: &MockExecutor{Output: []byte(output)}}
	results, err := n.Search("neovim")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestNixBackend_Search_ExecError(t *testing.T) {
	n := &NixBackend{exec: &MockExecutor{Err: errors.New("search failed")}}
	if _, err := n.Search("neovim"); err == nil {
		t.Error("expected error, got nil")
	}
}

// --- FlatpakBackend ---

func TestFlatpakBackend_Install_EmptyName(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{}}
	if err := f.Install(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestFlatpakBackend_Install_Success(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{}}
	if err := f.Install("com.spotify.Client"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFlatpakBackend_Install_ExecError(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{Err: errors.New("flatpak: not found")}}
	if err := f.Install("com.spotify.Client"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestFlatpakBackend_Remove_EmptyName(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{}}
	if err := f.Remove(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestFlatpakBackend_Remove_Success(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{}}
	if err := f.Remove("com.spotify.Client"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFlatpakBackend_Remove_ExecError(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{Err: errors.New("remove failed")}}
	if err := f.Remove("com.spotify.Client"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestFlatpakBackend_Update_EmptyName(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{}}
	if err := f.Update(""); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestFlatpakBackend_Update_Success(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{}}
	if err := f.Update("com.spotify.Client"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFlatpakBackend_Update_ExecError(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{Err: errors.New("update failed")}}
	if err := f.Update("com.spotify.Client"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestFlatpakBackend_UpdateAll_Success(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{}}
	if err := f.UpdateAll(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFlatpakBackend_UpdateAll_ExecError(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{Err: errors.New("update failed")}}
	if err := f.UpdateAll(); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestFlatpakBackend_List_Success(t *testing.T) {
	output := "Application\tName\tDescription\ncom.spotify.Client\tSpotify\tMusic streaming\norg.mozilla.firefox\tFirefox\tWeb browser"
	f := &FlatpakBackend{exec: &MockExecutor{Output: []byte(output)}}
	results, err := f.List()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (excluding header), got %d", len(results))
	}
}

func TestFlatpakBackend_List_ExecError(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{Err: errors.New("list failed")}}
	if _, err := f.List(); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestFlatpakBackend_Search_EmptyQuery(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{}}
	if _, err := f.Search(""); err == nil {
		t.Error("expected error for empty query")
	}
}

func TestFlatpakBackend_Search_Success(t *testing.T) {
	output := "Application\tName\tDescription\ncom.spotify.Client\tSpotify\tMusic streaming\norg.mozilla.firefox\tFirefox\tWeb browser"
	f := &FlatpakBackend{exec: &MockExecutor{Output: []byte(output)}}
	results, err := f.Search("firefox")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (sem header), got %d", len(results))
	}
}

func TestFlatpakBackend_Search_ExecError(t *testing.T) {
	f := &FlatpakBackend{exec: &MockExecutor{Err: errors.New("search failed")}}
	if _, err := f.Search("firefox"); err == nil {
		t.Error("expected error, got nil")
	}
}

// --- AgnosticManager ---

func TestNewAgnosticManager_HasAllBackends(t *testing.T) {
	mgr := NewAgnosticManager()
	// Todos os backends agora são registrados condicionalmente com base
	// na presença do binário no PATH.
	backends := map[string]string{
		"pacman":  "pacman",
		"nix":     "nix",
		"flatpak": "flatpak",
		"apt":     "apt-get",
		"dnf":     "dnf",
		"yum":     "yum",
		"zypper":  "zypper",
		"brew":    "brew",
	}
	for name, bin := range backends {
		if _, err := exec.LookPath(bin); err == nil {
			if _, ok := mgr.Backends[name]; !ok {
				t.Errorf("expected backend '%s' to be registered when '%s' is in PATH", name, bin)
			}
		}
	}
}

func TestAgnosticManager_RegisterBackend(t *testing.T) {
	mgr := NewAgnosticManager()
	mgr.RegisterBackend("mock", &MockBackend{})
	if _, ok := mgr.Backends["mock"]; !ok {
		t.Error("expected mock backend to be registered")
	}
}

func TestAgnosticManager_ListBackends(t *testing.T) {
	mgr := NewAgnosticManager()
	list := mgr.ListBackends()
	// ListBackends deve retornar exatamente os backends registrados
	// (que agora são condicionais com base na presença do binário).
	if len(list) != len(mgr.Backends) {
		t.Errorf("expected %d backends, got %d", len(mgr.Backends), len(list))
	}
	// Verifica que não há duplicatas
	seen := make(map[string]bool)
	for _, name := range list {
		if seen[name] {
			t.Errorf("duplicate backend in list: %s", name)
		}
		seen[name] = true
	}
	// Verifica que todo backend na lista existe no map
	for name := range mgr.Backends {
		if !seen[name] {
			t.Errorf("backend %s is in Backends map but not in ListBackends()", name)
		}
	}
}

// MockBackend implementa PackageService para testes de manager
type MockBackend struct {
	InstallErr   error
	RemoveErr    error
	UpdateErr    error
	UpdateAllErr error
	SearchRes    []string
	SearchErr    error
}

func (m *MockBackend) Install(pkgName string) error            { return m.InstallErr }
func (m *MockBackend) Remove(pkgName string) error             { return m.RemoveErr }
func (m *MockBackend) Update(pkg string) error                 { return m.UpdateErr }
func (m *MockBackend) UpdateAll() error                        { return m.UpdateAllErr }
func (m *MockBackend) Search(q string) ([]string, error)       { return m.SearchRes, m.SearchErr }
func (m *MockBackend) List() ([]string, error)                 { return []string{"pkg1", "pkg2"}, nil }
