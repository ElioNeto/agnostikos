package manager

import (
	"context"
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
)

// PackageService define o contrato para qualquer gerenciador de pacotes externo
type PackageService interface {
	Install(pkgName string) error
	Remove(pkgName string) error
	Update(pkg string) error
	UpdateAll() error
	Search(query string) ([]string, error)
	List() ([]string, error) // lista pacotes instalados
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

// BuildConfig contém os parâmetros para a construção completa da ISO AgnosticOS.
type BuildConfig struct {
	TargetDir     string
	KernelVersion string
	Arch          string
	UEFI          bool
	Jobs          string
	SkipToolchain bool
	SkipKernel    bool
	SkipBusybox   bool
	SkipInitramfs bool
	SkipGRUB      bool
	Force         bool
	OutputISO     string
	Name          string
	Version       string
	BootLabel     string
}

// Build executa o pipeline completo de bootstrap e geração de ISO.
func (m *AgnosticManager) Build(ctx context.Context, cfg BuildConfig) error {
	bootstrapCfg := bootstrap.BootstrapConfig{
		TargetDir:      cfg.TargetDir,
		KernelVersion:  cfg.KernelVersion,
		Arch:           cfg.Arch,
		UEFI:           cfg.UEFI,
		Jobs:           cfg.Jobs,
		SkipToolchain:  cfg.SkipToolchain,
		SkipKernel:     cfg.SkipKernel,
		SkipBusybox:    cfg.SkipBusybox,
		SkipInitramfs:  cfg.SkipInitramfs,
		SkipGRUB:       cfg.SkipGRUB,
		Force:          cfg.Force,
	}

	if err := bootstrap.BootstrapAll(ctx, bootstrapCfg); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	target := cfg.TargetDir
	if target == "" {
		target = bootstrap.DefaultRoot
	}

	isoOut := cfg.OutputISO
	if isoOut == "" {
		isoOut = bootstrap.BaseDir + "/build/agnostikos-latest.iso"
	}

	name := cfg.Name
	if name == "" {
		name = "AgnostikOS"
	}

	version := cfg.Version
	if version == "" {
		version = "0.1.0"
	}

	bootLabel := cfg.BootLabel
	if bootLabel == "" {
		bootLabel = name + " " + version
	}

	isoCfg := bootstrap.ISOConfig{
		Name:          name,
		Version:       version,
		KernelVersion: cfg.KernelVersion,
		RootFS:        target,
		Output:        isoOut,
		UEFI:          cfg.UEFI,
		BootLabel:     bootLabel,
	}

	if err := bootstrap.GenerateISO(isoCfg); err != nil {
		return fmt.Errorf("ISO generation failed: %w", err)
	}

	return nil
}
