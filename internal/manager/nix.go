package manager

import "fmt"

// NixBackend implementa PackageService usando nix (stub)
type NixBackend struct{}

func (n *NixBackend) Install(pkgName string) error {
	return fmt.Errorf("nix backend: not yet implemented")
}
func (n *NixBackend) Remove(pkgName string) error {
	return fmt.Errorf("nix backend: not yet implemented")
}
func (n *NixBackend) Update() error {
	return fmt.Errorf("nix backend: not yet implemented")
}
func (n *NixBackend) Search(query string) ([]string, error) {
	return nil, fmt.Errorf("nix backend: not yet implemented")
}
