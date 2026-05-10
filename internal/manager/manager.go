package manager

import (
	"context"
	"fmt"
	"os/exec"

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
	Resolver Resolver
}

// NewAgnosticManager inicializa o manager com todos os backends registrados
func NewAgnosticManager() *AgnosticManager {
	backends := map[string]PackageService{
		"pacman":  NewPacmanBackend(),
		"nix":     NewNixBackend(),
		"flatpak": NewFlatpakBackend(),
	}

	// Registra APT backend se apt-get estiver disponível no PATH
	if _, err := exec.LookPath("apt-get"); err == nil {
		backends["apt"] = NewAptBackend()
	}

	// Registra DNF/YUM backend se dnf ou yum estiver disponível
	if _, err := exec.LookPath("dnf"); err == nil {
		backends["dnf"] = NewDNFBackend()
	} else if _, err := exec.LookPath("yum"); err == nil {
		backends["yum"] = NewDNFBackend()
	}

	// Registra Zypper backend se zypper estiver disponível
	if _, err := exec.LookPath("zypper"); err == nil {
		backends["zypper"] = NewZypperBackend()
	}

	// Registra Homebrew backend se brew estiver disponível
	if _, err := exec.LookPath("brew"); err == nil {
		backends["brew"] = NewBrewBackend()
	}

	return &AgnosticManager{
		Backends: backends,
		Resolver: NewResolver(backends),
	}
}

// WithResolver configura um Resolver customizado (útil para testes).
func WithResolver(r Resolver) func(*AgnosticManager) {
	return func(m *AgnosticManager) {
		m.Resolver = r
	}
}

// ResolvePackage resolves which backend should handle a package based on the given policy.
// It returns a ResolveResult with the selected backend and version info.
func (m *AgnosticManager) ResolvePackage(ctx context.Context, pkg string, policy ResolvePolicy) (ResolveResult, error) {
	if m.Resolver == nil {
		return ResolveResult{}, fmt.Errorf("resolver not initialized")
	}
	return m.Resolver.Resolve(ctx, pkg, policy)
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
	TargetDir      string `json:"target_dir,omitempty"`
	BusyboxVersion string `json:"busybox_version,omitempty"`
	Device         string `json:"device,omitempty"`
	EFIPartition   string `json:"efi_partition,omitempty"`
	KernelVersion  string `json:"kernel_version,omitempty"`
	Arch           string `json:"arch,omitempty"`
	UEFI           bool   `json:"uefi"`
	Jobs           string `json:"jobs,omitempty"`
	SkipToolchain  bool   `json:"skip_toolchain"`
	SkipKernel     bool   `json:"skip_kernel"`
	SkipBusybox    bool   `json:"skip_busybox"`
	SkipInitramfs  bool   `json:"skip_initramfs"`
	SkipGRUB       bool   `json:"skip_grub"`
	Force          bool   `json:"force"`
	OutputISO      string `json:"output_iso,omitempty"`
	Name           string `json:"name,omitempty"`
	Version        string `json:"version,omitempty"`
	BootLabel      string `json:"boot_label,omitempty"`
}

// Build executa o pipeline completo de bootstrap e geração de ISO.
func (m *AgnosticManager) Build(ctx context.Context, cfg BuildConfig) error {
	busyboxVersion := cfg.BusyboxVersion
	if busyboxVersion == "" {
		busyboxVersion = "1.36.1"
	}

	bootstrapCfg := bootstrap.BootstrapConfig{
		TargetDir:      cfg.TargetDir,
		BusyboxVersion: busyboxVersion,
		Device:         cfg.Device,
		EFIPartition:   cfg.EFIPartition,
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
