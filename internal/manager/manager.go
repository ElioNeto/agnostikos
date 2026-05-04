package manager

// PackageService define o contrato para qualquer gerenciador de pacotes externo
type PackageService interface {
	Install(pkgName string) error
	Remove(pkgName string) error
	Update() error
	Search(query string) ([]string, error)
}

// AgnosticManager coordena os múltiplos backends
type AgnosticManager struct {
	Backends map[string]PackageService
}

// NewAgnosticManager inicializa o manager com todos os backends registrados
func NewAgnosticManager() *AgnosticManager {
	return &AgnosticManager{
		Backends: map[string]PackageService{
			"pacman":  NewPacmanBackend(),
			"nix":     NewNixBackend(),
			"flatpak": NewFlatpakBackend(),
		},
	}
}

// RegisterBackend adiciona um backend customizado em runtime
func (m *AgnosticManager) RegisterBackend(name string, b PackageService) {
	m.Backends[name] = b
}

// ListBackends retorna os nomes dos backends disponíveis
func (m *AgnosticManager) ListBackends() []string {
	keys := make([]string, 0, len(m.Backends))
	for k := range m.Backends {
		keys = append(keys, k)
	}
	return keys
}
