package bootstrap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ElioNeto/agnostikos/internal/dotfiles"
)

// BaseDir é o diretório raiz de todo o ambiente AgnosticOS.
// Tudo relacionado a build, rootfs, sources e ISO fica aqui dentro.
const BaseDir = "/mnt/data/agnostikOS"

// DefaultRoot é o diretório padrão do RootFS dentro do BaseDir.
const DefaultRoot = BaseDir + "/rootfs"

// ToolchainPackage descreve um pacote da toolchain
type ToolchainPackage struct {
	Name string
	URL  string
}

// DefaultToolchain lista os pacotes base
var DefaultToolchain = []ToolchainPackage{
	{"binutils-2.42", "https://sourceware.org/pub/binutils/releases/binutils-2.42.tar.xz"},
	{"gcc-14.1.0", "https://ftp.gnu.org/gnu/gcc/gcc-14.1.0/gcc-14.1.0.tar.xz"},
	{"glibc-2.39", "https://ftp.gnu.org/gnu/glibc/glibc-2.39.tar.xz"},
}

// FHSDirectories é a árvore de diretórios do Filesystem Hierarchy Standard
var FHSDirectories = []string{
	"boot",
	"dev",
	"etc",
	"home",
	"media",
	"mnt",
	"opt",
	"proc",
	"root",
	"run",
	"srv",
	"sys",
	"tmp",
	"usr/bin",
	"usr/lib",
	"usr/sbin",
	"usr/include",
	"usr/share",
	"usr/local",
	"usr/src",
	"var/cache",
	"var/lib",
	"var/log",
	"var/run",
	"var/tmp",
}

// resolveTarget retorna o target resolvido: arg > env AGNOSTICOS_ROOT > DefaultRoot
func resolveTarget(target string) string {
	if target != "" {
		return target
	}
	if v := os.Getenv("AGNOSTICOS_ROOT"); v != "" {
		return v
	}
	return DefaultRoot
}

// sourcesDir retorna /mnt/data/agnostikOS/sources (ou base derivado de rootfsDir)
func sourcesDir(rootfsDir string) string {
	base := BaseDir
	if rootfsDir != "" && rootfsDir != DefaultRoot {
		base = filepath.Dir(rootfsDir)
	} else if v := os.Getenv("AGNOSTICOS_ROOT"); v != "" {
		base = filepath.Dir(v)
	}
	return filepath.Join(base, "sources")
}

// tmpDir retorna um diretório temporário sob o temp dir do sistema
func tmpDir() string {
	return filepath.Join(os.TempDir(), "agnostikos-tmp")
}

// artifactExists retorna true se o caminho dado já existir no disco.
func artifactExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CreateRootFS monta a árvore FHS no diretório alvo e inicializa o VirtualFS
func CreateRootFS(target string) error {
	target = resolveTarget(target)
	fmt.Printf("[rootfs] Creating RootFS at: %s\n", target)

	for _, dir := range FHSDirectories {
		path := filepath.Join(target, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}
	}

	symlinks := map[string]string{
		filepath.Join(target, "bin"):     "usr/bin",
		filepath.Join(target, "lib"):     "usr/lib",
		filepath.Join(target, "lib64"):   "usr/lib",
		filepath.Join(target, "sbin"):    "usr/sbin",
		filepath.Join(target, "var/run"): "../run",
	}
	for link, dest := range symlinks {
		_ = os.Remove(link)
		if err := os.Symlink(dest, link); err != nil {
			fmt.Printf("[rootfs] warn: symlink %s -> %s: %v\n", link, dest, err)
		}
	}

	fmt.Println("[rootfs] FHS structure created")
	return mountVirtualFS(target)
}

// DownloadToolchain baixa os pacotes da toolchain para /mnt/data/agnostikOS/sources
func DownloadToolchain(rootfsDir string) error {
	rootfsDir = resolveTarget(rootfsDir)
	src := sourcesDir(rootfsDir)
	if err := os.MkdirAll(src, 0755); err != nil {
		return fmt.Errorf("mkdir sources: %w", err)
	}
	for _, pkg := range DefaultToolchain {
		dest := filepath.Join(src, filepath.Base(pkg.URL))
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("[toolchain] already exists: %s\n", pkg.Name)
			continue
		}
		fmt.Printf("[toolchain] downloading %s...\n", pkg.Name)
		if err := downloadFile(dest, pkg.URL); err != nil {
			return fmt.Errorf("download %s: %w", pkg.Name, err)
		}
		fmt.Printf("[toolchain] downloaded %s\n", pkg.Name)
	}
	return nil
}

// httpClient is a variable so tests can replace it with a mock.
var httpClient = http.DefaultClient

// downloadFile faz o download de uma URL para um arquivo local
func downloadFile(dest, url string) error {
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// mountVirtualFS monta proc/sys/dev dentro do chroot
func mountVirtualFS(target string) error {
	type mountSpec struct {
		fstype, source, target, opts string
	}
	mounts := []mountSpec{
		{"proc", "proc", filepath.Join(target, "proc"), ""},
		{"sysfs", "sysfs", filepath.Join(target, "sys"), ""},
		{"devtmpfs", "devtmpfs", filepath.Join(target, "dev"), "mode=0755"},
		{"devpts", "devpts", filepath.Join(target, "dev/pts"), "gid=5,mode=0620"},
		{"tmpfs", "tmpfs", filepath.Join(target, "run"), ""},
	}
	for _, m := range mounts {
		args := []string{"-t", m.fstype}
		if m.opts != "" {
			args = append(args, "-o", m.opts)
		}
		args = append(args, m.source, m.target)
		if out, err := exec.CommandContext(context.Background(), "mount", args...).CombinedOutput(); err != nil {
			fmt.Printf("[rootfs] warn: mount %s: %s\n", m.fstype, string(out))
		} else {
			fmt.Printf("[rootfs] mounted %s -> %s\n", m.fstype, m.target)
		}
	}

	return nil
}

// UnmountVirtualFS desmonta os filesystems virtuais do chroot
func UnmountVirtualFS(target string) error {
	for _, p := range []string{"dev/pts", "dev", "run", "proc", "sys"} {
		_ = exec.CommandContext(context.Background(), "umount", filepath.Join(target, p)).Run()
	}
	return nil
}

// BootstrapConfig contém todos os parâmetros para a construção completa do RootFS
type BootstrapConfig struct {
	TargetDir      string // diretório raiz do RootFS (ex: /mnt/data/agnostikOS/rootfs)
	Device         string // disco base para grub-install BIOS (ex: /dev/sda)
	EFIPartition   string // partição ESP para grub-install UEFI (ex: /dev/nvme0n1p1)
	KernelVersion  string // versão do kernel Linux (ex: "6.6")
	BusyboxVersion string // versão do Busybox (ex: "1.36.1")
	Arch           string // target arch: "amd64" ou "arm64" (vazio = auto-detect de runtime.GOARCH)
	UEFI           bool   // gerar estrutura UEFI
	SkipToolchain  bool   // pular compilação da toolchain (binutils, gcc, glibc)
	SkipKernel     bool   // pular compilação do kernel
	SkipBusybox    bool   // pular compilação do busybox
	SkipInitramfs  bool   // pular geração do initramfs
	SkipGRUB       bool   // pular instalação do GRUB
	Force          bool   // ignorar cache e recompilar tudo
	Jobs           string // número de jobs paralelos para make -j (vazio = auto, max 4)
	DotfilesApply  bool   // aplicar dotfiles ao final do bootstrap
	DotfilesSource string // URL git ou caminho local para dotfiles externos
	ConfigsDir     string // diretório dos dotfiles embutidos (configs/)
	AutoLoginUser  string   // usuário para autologin via getty (vazio = desabilitado)
	MiseRuntimes   []string // runtimes to install via mise (e.g. ["nodejs@lts", "python@3", "ruby", "java"])
}

// kernelImageName retorna o nome do arquivo da imagem do kernel de acordo com a arquitetura.
func kernelImageName(arch, version string) string {
	if arch == "arm64" {
		return "Image-" + version
	}
	return "vmlinuz-" + version
}

// configureDefaultShell configura Zsh como shell padrão no rootfs.
// É idempotente: verifica se /bin/zsh existe, registra em /etc/shells
// e atualiza a entrada do root em /etc/passwd.
func configureDefaultShell(rootfsDir string) error {
	zshBin := filepath.Join(rootfsDir, "/bin/zsh")
	if _, err := os.Stat(zshBin); os.IsNotExist(err) {
		fmt.Printf("[shell] /bin/zsh not found in rootfs, skipping default shell config\n")
		return nil
	}

	// /etc/shells
	shellsPath := filepath.Join(rootfsDir, "etc", "shells")
	if err := os.MkdirAll(filepath.Dir(shellsPath), 0755); err != nil {
		return fmt.Errorf("mkdir /etc: %w", err)
	}
	data, err := os.ReadFile(shellsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read /etc/shells: %w", err)
	}
	if !hasShellEntry(string(data), "/bin/zsh") {
		f, err := os.OpenFile(shellsPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("open /etc/shells: %w", err)
		}
		defer func() { _ = f.Close() }()
		if _, err := fmt.Fprintln(f, "/bin/zsh"); err != nil {
			return fmt.Errorf("write /etc/shells: %w", err)
		}
		fmt.Printf("[shell] added /bin/zsh to %s\n", shellsPath)
	}

	// /etc/passwd — set root's shell to /bin/zsh
	passwdPath := filepath.Join(rootfsDir, "etc", "passwd")
	if err := os.MkdirAll(filepath.Dir(passwdPath), 0755); err != nil {
		return fmt.Errorf("mkdir /etc: %w", err)
	}
	passwdData, err := os.ReadFile(passwdPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read /etc/passwd: %w", err)
	}
	lines := strings.Split(string(passwdData), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.Split(trimmed, ":")
		if len(parts) >= 7 && parts[0] == "root" {
			parts[6] = "/bin/zsh"
			lines[i] = strings.Join(parts, ":")
			found = true
			break
		}
	}
	if !found {
		// No root entry exists; append one
		lines = append(lines, "root:x:0:0:root:/root:/bin/zsh")
	}
	updated := strings.Join(lines, "\n")
	if updated != string(passwdData) {
		if err := os.WriteFile(passwdPath, []byte(updated), 0644); err != nil {
			return fmt.Errorf("write /etc/passwd: %w", err)
		}
		fmt.Printf("[shell] updated root shell to /bin/zsh in %s\n", passwdPath)
	}

	return nil
}

// hasShellEntry verifica se um caminho de shell já existe no conteúdo de /etc/shells.
func hasShellEntry(content, shell string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == shell {
			return true
		}
	}
	return false
}

// configureAutologin configura o autologin automático no tty1 via systemd getty.
// Cria um drop-in em /etc/systemd/system/getty@tty1.service.d/autologin.conf
// com ExecStart apontando para agetty --autologin <username>.
func configureAutologin(rootfsDir, username string) error {
	if username == "" {
		return nil
	}

	dropinDir := filepath.Join(rootfsDir, "etc", "systemd", "system", "getty@tty1.service.d")
	if err := os.MkdirAll(dropinDir, 0755); err != nil {
		return fmt.Errorf("mkdir getty drop-in: %w", err)
	}

	content := fmt.Sprintf(`# Auto-login for %s — managed by AgnosticOS
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin %s --noclear %%I $TERM
`, username, username)

	dropinPath := filepath.Join(dropinDir, "autologin.conf")
	if err := os.WriteFile(dropinPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write autologin.conf: %w", err)
	}

	fmt.Printf("[autologin] configured auto-login for user %s on tty1\n", username)
	return nil
}

// setupMiseRuntimes instala runtimes via mise no rootfs.
// Verifica se o binário mise existe e executa mise install para cada runtime listado.
func setupMiseRuntimes(rootfsDir string, runtimes []string) {
	if len(runtimes) == 0 {
		return
	}

	miseBin := filepath.Join(rootfsDir, "/usr/bin/mise")
	if _, err := os.Stat(miseBin); os.IsNotExist(err) {
		fmt.Printf("[mise] /usr/bin/mise not found in rootfs, skipping runtime install\n")
		return
	}

	fmt.Printf("[mise] Installing %d runtimes: %s\n", len(runtimes), strings.Join(runtimes, ", "))

	// Write a profile.d script for mise activation
	profileScript := `# mise activation — managed by AgnosticOS
if command -v mise &>/dev/null; then
  eval "$(mise activate zsh)"
fi
`
	profileDir := filepath.Join(rootfsDir, "etc", "profile.d")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		fmt.Printf("[mise] warn: mkdir /etc/profile.d: %v\n", err)
		return
	}
	profilePath := filepath.Join(profileDir, "mise.sh")
	if err := os.WriteFile(profilePath, []byte(profileScript), 0644); err != nil {
		fmt.Printf("[mise] warn: write mise.sh: %v\n", err)
		return
	}
	fmt.Printf("[mise] wrote activation script to %s\n", profilePath)

	// Run mise install for each runtime
	for _, rt := range runtimes {
		fmt.Printf("[mise] Installing %s...\n", rt)
		cmd := exec.Command(miseBin, "install", rt)
		cmd.Env = append(os.Environ(), "MISE_YES=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("[mise] warn: install %s failed: %v\n%s\n", rt, err, string(output))
		} else {
			fmt.Printf("[mise] Installed %s\n", rt)
		}
	}
}

// BootstrapAll executa o pipeline completo de construção do RootFS
func BootstrapAll(ctx context.Context, cfg BootstrapConfig) error {
	if cfg.TargetDir == "" {
		cfg.TargetDir = resolveTarget("")
	}

	fmt.Printf("[bootstrap] Starting full bootstrap at %s\n", cfg.TargetDir)
	arch := cfg.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	fmt.Printf("[bootstrap] Config: kernel=%s busybox=%s arch=%s uefi=%v force=%v jobs=%s\n",
		cfg.KernelVersion, cfg.BusyboxVersion, arch, cfg.UEFI, cfg.Force, cfg.Jobs)

	// Step 1: RootFS — idempotente, MkdirAll é no-op se já existe
	fmt.Println("\n=== Step 1/13: Create RootFS ===")
	if err := CreateRootFS(cfg.TargetDir); err != nil {
		return fmt.Errorf("create rootfs: %w", err)
	}

	tcCfg := ToolchainConfig{
		TargetDir: cfg.TargetDir,
		NumCPUs:   cfg.Jobs,
	}

	// Step 2: Toolchain — download dos tarballs
	if !cfg.SkipToolchain {
		fmt.Println("\n=== Step 2/13: Download Toolchain ===")
		if err := DownloadToolchain(cfg.TargetDir); err != nil {
			return fmt.Errorf("download toolchain: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 2/13: Download Toolchain (skipped) ===")
	}

	// Step 3: Build binutils
	if !cfg.SkipToolchain {
		fmt.Println("\n=== Step 3/13: Build binutils ===")
		if err := BuildBinutils(ctx, tcCfg); err != nil {
			return fmt.Errorf("build binutils: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 3/13: Build binutils (skipped) ===")
	}

	// Step 4: Build GCC (pass 1, C only)
	if !cfg.SkipToolchain {
		fmt.Println("\n=== Step 4/13: Build GCC (pass 1) ===")
		if err := BuildGCC(ctx, tcCfg); err != nil {
			return fmt.Errorf("build gcc: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 4/13: Build GCC (skipped) ===")
	}

	// Step 5: Build glibc
	if !cfg.SkipToolchain {
		fmt.Println("\n=== Step 5/13: Build glibc ===")
		if err := BuildGLibc(ctx, tcCfg); err != nil {
			return fmt.Errorf("build glibc: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 5/13: Build glibc (skipped) ===")
	}

	// Step 6: Kernel
	kernelImage := kernelImageName(arch, cfg.KernelVersion)
	kernelArtifact := filepath.Join(cfg.TargetDir, "boot", kernelImage)
	if !cfg.SkipKernel {
		if !cfg.Force && artifactExists(kernelArtifact) {
			fmt.Printf("\n=== Step 6/13: Build Kernel (cached: %s) ===\n", kernelArtifact)
		} else {
			fmt.Println("\n=== Step 6/13: Build Kernel ===")
			kernelCfg := KernelConfig{
				Version:    cfg.KernelVersion,
				SourcesDir: sourcesDir(cfg.TargetDir),
				OutputDir:  filepath.Join(cfg.TargetDir, "boot"),
				Defconfig:  "", // auto-detect from arch
				Arch:       arch,
			}
			if err := BuildKernel(kernelCfg); err != nil {
				return fmt.Errorf("build kernel: %w", err)
			}
		}
	} else {
		fmt.Println("\n=== Step 6/13: Build Kernel (skipped by flag) ===")
	}

	// Step 7: Busybox
	busyboxArtifact := filepath.Join(cfg.TargetDir, "busybox-install", "bin", "busybox")
	if !cfg.SkipBusybox {
		if !cfg.Force && artifactExists(busyboxArtifact) {
			fmt.Printf("\n=== Step 7/13: Build Busybox (cached: %s) ===\n", busyboxArtifact)
		} else {
			fmt.Println("\n=== Step 7/13: Build Busybox ===")
			busyboxCfg := BusyboxConfig{
				Version:   cfg.BusyboxVersion,
				TargetDir: cfg.TargetDir,
			}
			if err := BuildBusybox(ctx, busyboxCfg); err != nil {
				return fmt.Errorf("build busybox: %w", err)
			}
		}
	} else {
		fmt.Println("\n=== Step 7/13: Build Busybox (skipped by flag) ===")
	}

	// Step 8: Initramfs
	initramfsArtifact := filepath.Join(cfg.TargetDir, "boot", "initramfs.img")
	if !cfg.SkipInitramfs {
		if !cfg.Force && artifactExists(initramfsArtifact) {
			fmt.Printf("\n=== Step 8/13: Build Initramfs (cached: %s) ===\n", initramfsArtifact)
		} else {
			fmt.Println("\n=== Step 8/13: Build Initramfs ===")
			outputDir := filepath.Join(cfg.TargetDir, "boot")
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("mkdir boot: %w", err)
			}
			if err := BuildInitramfs(ctx, cfg.TargetDir, initramfsArtifact); err != nil {
				return fmt.Errorf("build initramfs: %w", err)
			}
		}
	} else {
		fmt.Println("\n=== Step 8/13: Build Initramfs (skipped by flag) ===")
	}

	// Step 9: GRUB
	grubArtifact := filepath.Join(cfg.TargetDir, "boot", "grub", "grub.cfg")
	if !cfg.SkipGRUB {
		if !cfg.Force && artifactExists(grubArtifact) {
			fmt.Printf("\n=== Step 9/13: Install GRUB (cached: %s) ===\n", grubArtifact)
		} else {
			fmt.Println("\n=== Step 9/13: Install GRUB ===")
			if err := InstallGRUB(ctx, GRUBConfig{
				RootfsDir:    cfg.TargetDir,
				Device:       cfg.Device,
				UEFI:         cfg.UEFI,
				EFIPartition: cfg.EFIPartition,
				Strict:       true,
			}); err != nil {
				return fmt.Errorf("install grub: %w", err)
			}
		}
	} else {
		fmt.Println("\n=== Step 9/13: Install GRUB (skipped by flag) ===")
	}

	// Step 10: Configure default shell (Zsh)
	fmt.Println("\n=== Step 10/13: Configure Default Shell ===")
	if err := configureDefaultShell(cfg.TargetDir); err != nil {
		return fmt.Errorf("configure default shell: %w", err)
	}

	// Step 11: Setup mise runtimes (optional)
	if len(cfg.MiseRuntimes) > 0 {
		fmt.Println("\n=== Step 11/13: Setup Mise Runtimes ===")
		setupMiseRuntimes(cfg.TargetDir, cfg.MiseRuntimes)
	} else {
		fmt.Println("\n=== Step 11/13: Setup Mise Runtimes (skipped) ===")
	}

	// Step 12: Configure autologin (optional)
	if cfg.AutoLoginUser != "" {
		fmt.Println("\n=== Step 12/13: Configure Autologin ===")
		if err := configureAutologin(cfg.TargetDir, cfg.AutoLoginUser); err != nil {
			return fmt.Errorf("configure autologin: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 12/13: Configure Autologin (skipped) ===")
	}

	// Step 13: Apply dotfiles (optional)
	if cfg.DotfilesApply {
		fmt.Println("\n=== Step 13/13: Apply Dotfiles ===")
		configsDir := cfg.ConfigsDir
		if configsDir == "" {
			configsDir = filepath.Join(filepath.Dir(os.Args[0]), "configs")
		}
		homeDir := filepath.Join(cfg.TargetDir, "root")
		mgr := dotfiles.New(cfg.DotfilesSource)
		if err := mgr.Apply(configsDir, homeDir, true); err != nil {
			return fmt.Errorf("apply dotfiles: %w", err)
		}
		fmt.Println("[dotfiles] applied to rootfs")
	} else {
		fmt.Println("\n=== Step 13/13: Apply Dotfiles (skipped) ===")
	}

	fmt.Printf("\n[bootstrap] ✅ Bootstrap complete at %s\n", cfg.TargetDir)
	return nil
}
