// Package dotfiles provides a manager for applying, listing, and diffing
// dotfiles (configuration files) from a configs/ directory to the user's
// home directory, with XDG-aware symlink placement.
package dotfiles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// dotfileEntry descreve um arquivo de dotfile: caminho de origem (relativo a configsDir)
// e caminho de destino (relativo a homeDir, com XDG).
type dotfileEntry struct {
	SourceRel string // relativo a configsDir, ex: "zsh/.zshrc"
	DestRel   string // relativo a homeDir, ex: ".zshrc" ou ".config/nvim/init.lua"
}

// defaultDotfiles é a lista de dotfiles gerenciados e seus destinos XDG.
var defaultDotfiles = []dotfileEntry{
	{SourceRel: "zsh/.zshrc", DestRel: ".zshrc"},
	{SourceRel: "zsh/.zshenv", DestRel: ".zshenv"},
	{SourceRel: "git/.gitconfig", DestRel: ".gitconfig"},
	{SourceRel: "git/.gitignore_global", DestRel: ".gitignore_global"},
	{SourceRel: "neovim/init.lua", DestRel: ".config/nvim/init.lua"},
	{SourceRel: "starship/starship.toml", DestRel: ".config/starship.toml"},
	{SourceRel: "alacritty/alacritty.toml", DestRel: ".config/alacritty/alacritty.toml"},
	{SourceRel: "tmux/.tmux.conf", DestRel: ".tmux.conf"},
	{SourceRel: "hyprland/hyprland.conf", DestRel: ".config/hypr/hyprland.conf"},
	{SourceRel: "waybar/config", DestRel: ".config/waybar/config"},
	{SourceRel: "waybar/style.css", DestRel: ".config/waybar/style.css"},
}

// Manager gerencia a aplicação e listagem de dotfiles.
type Manager struct {
	// From é uma URL git opcional para clonar dotfiles externos.
	From string
}

// New cria um novo Manager. Se from não for vazio, será usado como URL git
// para clonar os dotfiles.
func New(from string) *Manager {
	return &Manager{From: from}
}

// Apply cria symlinks dos dotfiles de configsDir para homeDir.
// Se force for true, sobrescreve arquivos existentes.
// Se force for false, faz backup dos arquivos existentes adicionando sufixo .bak.
func (m *Manager) Apply(configsDir, homeDir string, force bool) error {
	// Se From estiver definido, clona o repositório externo primeiro.
	if m.From != "" {
		cloned, err := m.cloneExternal(configsDir)
		if err != nil {
			return fmt.Errorf("clone external dotfiles: %w", err)
		}
		configsDir = cloned
	}

	for _, entry := range defaultDotfiles {
		src := filepath.Join(configsDir, entry.SourceRel)
		dst := filepath.Join(homeDir, entry.DestRel)

		// Verifica se a origem existe
		if _, err := os.Stat(src); os.IsNotExist(err) {
			fmt.Printf("[dotfiles] warning: source not found, skipping: %s\n", src)
			continue
		} else if err != nil {
			return fmt.Errorf("stat source %s: %w", src, err)
		}

		// Cria diretório pai do destino
		dstDir := filepath.Dir(dst)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dstDir, err)
		}

		// Verifica se o destino já existe
		if _, err := os.Lstat(dst); err == nil {
			// Já existe algo no destino
			if force {
				// Remove para substituir pelo symlink
				if err := os.RemoveAll(dst); err != nil {
					return fmt.Errorf("remove existing %s: %w", dst, err)
				}
				fmt.Printf("[dotfiles] removed existing %s (force)\n", dst)
			} else {
				// Faz backup
				backup := dst + ".bak"
				if err := os.Rename(dst, backup); err != nil {
					return fmt.Errorf("backup %s -> %s: %w", dst, backup, err)
				}
				fmt.Printf("[dotfiles] backed up %s -> %s\n", dst, backup)
			}
		}

		// Cria o symlink (relativo para portabilidade)
		relLink, err := filepath.Rel(dstDir, src)
		if err != nil {
			return fmt.Errorf("relative path %s -> %s: %w", dstDir, src, err)
		}
		if err := os.Symlink(relLink, dst); err != nil {
			return fmt.Errorf("symlink %s -> %s: %w", dst, relLink, err)
		}
		fmt.Printf("[dotfiles] linked %s -> %s\n", dst, relLink)
	}
	return nil
}

// List retorna uma lista de dotfiles disponíveis em configsDir.
// Retorna os caminhos relativos a configsDir.
func (m *Manager) List(configsDir string) ([]string, error) {
	var available []string

	for _, entry := range defaultDotfiles {
		src := filepath.Join(configsDir, entry.SourceRel)
		if _, err := os.Stat(src); err == nil {
			available = append(available, entry.SourceRel)
		}
	}

	sort.Strings(available)
	return available, nil
}

// Diff compara os dotfiles em configsDir com os existentes em homeDir.
// Retorna uma lista de descrições das diferenças encontradas.
// Cada entrada descreve se o arquivo está faltando, diferente ou ok.
func (m *Manager) Diff(configsDir, homeDir string) ([]string, error) {
	var diffs []string

	for _, entry := range defaultDotfiles {
		src := filepath.Join(configsDir, entry.SourceRel)
		dst := filepath.Join(homeDir, entry.DestRel)

		// Verifica se a origem existe em configsDir
		srcInfo, err := os.Stat(src)
		if os.IsNotExist(err) {
			diffs = append(diffs, "MISSING (source): "+entry.SourceRel)
			continue
		} else if err != nil {
			return nil, fmt.Errorf("stat source %s: %w", src, err)
		}

		// Verifica destino
		dstInfo, err := os.Stat(dst)
		if os.IsNotExist(err) {
			diffs = append(diffs, fmt.Sprintf("MISSING (dest): %s -> %s", entry.SourceRel, entry.DestRel))
			continue
		} else if err != nil {
			return nil, fmt.Errorf("stat dest %s: %w", dst, err)
		}

		// Compara tamanho e mod time (aproximação)
		if srcInfo.Size() != dstInfo.Size() || !srcInfo.ModTime().Equal(dstInfo.ModTime()) {
			diffs = append(diffs, fmt.Sprintf("DIFFERENT: %s (src) vs %s (dest)", entry.SourceRel, entry.DestRel))
			continue
		}

		diffs = append(diffs, fmt.Sprintf("OK: %s -> %s", entry.SourceRel, entry.DestRel))
	}

	return diffs, nil
}

// cloneExternal clona um repositório git externo em um diretório temporário
// e retorna o caminho para o diretório clonado.
func (m *Manager) cloneExternal(destDir string) (string, error) {
	if m.From == "" {
		return "", errors.New("no source URL set")
	}

	// Extrai o nome do repositório da URL para criar um subdiretório
	repoName := filepath.Base(m.From)
	repoName = strings.TrimSuffix(repoName, ".git")
	cloneDir := filepath.Join(destDir, repoName)

	// Se já existe, remove para clonar fresco
	if _, err := os.Stat(cloneDir); err == nil {
		if err := os.RemoveAll(cloneDir); err != nil {
			return "", fmt.Errorf("remove existing clone dir: %w", err)
		}
	}

	cmd := exec.CommandContext(context.Background(), "git", "clone", "--depth=1", m.From, cloneDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone %s: %w\n%s", m.From, err, string(output))
	}

	fmt.Printf("[dotfiles] cloned %s -> %s\n", m.From, cloneDir)
	return cloneDir, nil
}
