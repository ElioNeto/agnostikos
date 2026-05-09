package integration

import (
	"strings"
	"sync"

	"github.com/ElioNeto/agnostikos/internal/manager"
)

// MockPackageService implements manager.PackageService with an in-memory
// stateful map for use in integration tests.
type MockPackageService struct {
	mu        sync.Mutex
	installed map[string]string // name -> version
}

// NewMockPackageService creates a new MockPackageService with an empty state.
func NewMockPackageService() *MockPackageService {
	return &MockPackageService{
		installed: make(map[string]string),
	}
}

// Install adds the package to the internal state with a fake version.
func (m *MockPackageService) Install(pkgName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installed[pkgName] = "1.0.0"
	return nil
}

// Remove deletes the package from the internal state.
func (m *MockPackageService) Remove(pkgName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.installed, pkgName)
	return nil
}

// Update is a no-op that returns nil to indicate success.
func (m *MockPackageService) Update(pkg string) error {
	return nil
}

// UpdateAll is a no-op that returns nil to indicate success.
func (m *MockPackageService) UpdateAll() error {
	return nil
}

// Search returns pre-defined known packages matching the query.
// It returns packages whose name contains the query string (case-sensitive).
func (m *MockPackageService) Search(query string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	known := []string{"htop", "curl", "base", "linux-firmware", "firefox", "git", "neovim", "tmux"}
	var results []string
	for _, pkg := range known {
		if strings.Contains(pkg, query) {
			results = append(results, pkg)
		}
	}
	return results, nil
}

// List returns all currently installed packages as "name version" strings.
func (m *MockPackageService) List() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]string, 0, len(m.installed))
	for name, version := range m.installed {
		result = append(result, name+" "+version)
	}
	return result, nil
}

// compile-time check that MockPackageService implements manager.PackageService
var _ manager.PackageService = (*MockPackageService)(nil)
