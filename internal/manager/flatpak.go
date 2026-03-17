package manager

import "fmt"

// FlatpakBackend implementa PackageService usando flatpak (stub)
type FlatpakBackend struct{}

func (f *FlatpakBackend) Install(pkgName string) error {
	return fmt.Errorf("flatpak backend: not yet implemented")
}
func (f *FlatpakBackend) Remove(pkgName string) error {
	return fmt.Errorf("flatpak backend: not yet implemented")
}
func (f *FlatpakBackend) Update() error {
	return fmt.Errorf("flatpak backend: not yet implemented")
}
func (f *FlatpakBackend) Search(query string) ([]string, error) {
	return nil, fmt.Errorf("flatpak backend: not yet implemented")
}
