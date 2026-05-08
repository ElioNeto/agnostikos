package dotfiles

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestDirs creates a temporary directory structure mimicking configs/ and home/.
// Returns configsDir, homeDir, and a cleanup function.
func setupTestDirs(t *testing.T) (configsDir, homeDir string, cleanup func()) {
	t.Helper()

	root := t.TempDir()
	configsDir = filepath.Join(root, "configs")
	homeDir = filepath.Join(root, "home")

	// Create config subdirectories
	dirs := []string{
		"zsh",
		"git",
		"neovim",
		"starship",
		"alacritty",
		"tmux",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(configsDir, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Create stub dotfiles in configs
	stubs := map[string]string{
		"zsh/.zshrc":              "export ZSH_PLUGINS=\"test\"\n",
		"zsh/.zshenv":             "export EDITOR=\"nvim\"\n",
		"git/.gitconfig":          "[user]\n\tname = Test\n",
		"git/.gitignore_global":   "*.log\n",
		"neovim/init.lua":         "vim.opt.number = true\n",
		"starship/starship.toml":  "format = \"test\"\n",
		"alacritty/alacritty.toml": "[font]\nsize = 12\n",
		"tmux/.tmux.conf":         "set -g mouse on\n",
	}
	for rel, content := range stubs {
		path := filepath.Join(configsDir, rel)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	cleanup = func() {
		os.RemoveAll(root)
	}

	return configsDir, homeDir, cleanup
}

func TestApply_CreatesSymlinks(t *testing.T) {
	configsDir, homeDir, cleanup := setupTestDirs(t)
	defer cleanup()

	mgr := New("")
	if err := mgr.Apply(configsDir, homeDir, false); err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}

	// Verify symlinks were created
	expected := map[string]string{
		".zshrc":                    "zsh/.zshrc",
		".zshenv":                   "zsh/.zshenv",
		".gitconfig":                "git/.gitconfig",
		".gitignore_global":         "git/.gitignore_global",
		".config/nvim/init.lua":     "neovim/init.lua",
		".config/starship.toml":     "starship/starship.toml",
		".config/alacritty/alacritty.toml": "alacritty/alacritty.toml",
		".tmux.conf":                "tmux/.tmux.conf",
	}

	for destRel, srcRel := range expected {
		linkPath := filepath.Join(homeDir, destRel)

		// Check symlink exists
		fi, err := os.Lstat(linkPath)
		if err != nil {
			t.Errorf("expected symlink %s to exist: %v", linkPath, err)
			continue
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", linkPath)
		}

		// Check symlink target
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Errorf("readlink %s: %v", linkPath, err)
			continue
		}

		// The symlink should be relative, pointing back to the source
		expectedTarget, err := filepath.Rel(filepath.Dir(linkPath), filepath.Join(configsDir, srcRel))
		if err != nil {
			t.Fatalf("relative path: %v", err)
		}
		if target != expectedTarget {
			t.Errorf("%s: expected target %q, got %q", linkPath, expectedTarget, target)
		}
	}
}

func TestApply_ExistingFileWithoutForce_BacksUp(t *testing.T) {
	configsDir, homeDir, cleanup := setupTestDirs(t)
	defer cleanup()

	// Create an existing file at the destination
	existingFile := filepath.Join(homeDir, ".zshrc")
	if err := os.MkdirAll(filepath.Dir(existingFile), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(existingFile, []byte("original content\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	mgr := New("")
	if err := mgr.Apply(configsDir, homeDir, false); err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}

	// Original should be backed up
	backupFile := existingFile + ".bak"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Errorf("expected backup file %s to exist", backupFile)
	} else if err != nil {
		t.Fatalf("stat backup: %v", err)
	}

	// Original path should now be a symlink
	fi, err := os.Lstat(existingFile)
	if err != nil {
		t.Fatalf("lstat %s: %v", existingFile, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("%s should be a symlink after apply", existingFile)
	}
}

func TestApply_ForceOverwritesExisting(t *testing.T) {
	configsDir, homeDir, cleanup := setupTestDirs(t)
	defer cleanup()

	// Create an existing file at the destination
	existingFile := filepath.Join(homeDir, ".zshenv")
	if err := os.MkdirAll(filepath.Dir(existingFile), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(existingFile, []byte("original\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	mgr := New("")
	if err := mgr.Apply(configsDir, homeDir, true); err != nil {
		t.Fatalf("Apply(force=true) returned error: %v", err)
	}

	// Backup should NOT exist
	backupFile := existingFile + ".bak"
	if _, err := os.Stat(backupFile); err == nil {
		t.Errorf("backup file %s should not exist with --force", backupFile)
	}

	// Original path should now be a symlink
	fi, err := os.Lstat(existingFile)
	if err != nil {
		t.Fatalf("lstat %s: %v", existingFile, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("%s should be a symlink after force apply", existingFile)
	}
}

func TestList_ReturnsExpectedEntries(t *testing.T) {
	configsDir, _, cleanup := setupTestDirs(t)
	defer cleanup()

	mgr := New("")
	list, err := mgr.List(configsDir)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	expected := []string{
		"alacritty/alacritty.toml",
		"git/.gitconfig",
		"git/.gitignore_global",
		"neovim/init.lua",
		"starship/starship.toml",
		"tmux/.tmux.conf",
		"zsh/.zshenv",
		"zsh/.zshrc",
	}

	if len(list) != len(expected) {
		t.Fatalf("List() returned %d entries, expected %d: %v", len(list), len(expected), list)
	}

	for i, item := range list {
		if item != expected[i] {
			t.Errorf("List()[%d] = %q, expected %q", i, item, expected[i])
		}
	}
}

func TestDiff_DetectsMissingFiles(t *testing.T) {
	configsDir, homeDir, cleanup := setupTestDirs(t)
	defer cleanup()

	mgr := New("")
	diffs, err := mgr.Diff(configsDir, homeDir)
	if err != nil {
		t.Fatalf("Diff() returned error: %v", err)
	}

	// All files should be "MISSING (dest)" since homeDir is empty
	for _, d := range diffs {
		if !contains(d, "MISSING (dest)") {
			t.Errorf("expected all entries to be MISSING (dest), got: %s", d)
		}
	}

	if len(diffs) != len(defaultDotfiles) {
		t.Errorf("expected %d diffs, got %d", len(defaultDotfiles), len(diffs))
	}
}

func TestDiff_DetectsChangedFiles(t *testing.T) {
	configsDir, homeDir, cleanup := setupTestDirs(t)
	defer cleanup()

	// Apply first to create symlinks
	mgr := New("")
	if err := mgr.Apply(configsDir, homeDir, false); err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}

	// Now everything should be OK
	diffs, err := mgr.Diff(configsDir, homeDir)
	if err != nil {
		t.Fatalf("Diff() returned error: %v", err)
	}

	for _, d := range diffs {
		if !contains(d, "OK:") {
			t.Errorf("expected all entries to be OK, got: %s", d)
		}
	}

	// Now modify one of the home files (write new content, break symlink by replacing with file)
	// Remove the symlink and create a real file with different content
	zshrcDest := filepath.Join(homeDir, ".zshrc")
	if err := os.Remove(zshrcDest); err != nil {
		t.Fatalf("remove symlink: %v", err)
	}
	if err := os.WriteFile(zshrcDest, []byte("modified content\n"), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	diffs2, err := mgr.Diff(configsDir, homeDir)
	if err != nil {
		t.Fatalf("Diff() returned error: %v", err)
	}

	foundDiff := false
	for _, d := range diffs2 {
		if contains(d, "DIFFERENT:") && contains(d, "zsh/.zshrc") {
			foundDiff = true
			break
		}
	}
	if !foundDiff {
		t.Error("expected Diff() to detect modified .zshrc, but no DIFFERENT entry found")
	}
}

func TestApply_MissingSource_DoesNotError(t *testing.T) {
	// Create a configs dir with no files
	root := t.TempDir()
	configsDir := filepath.Join(root, "configs")
	homeDir := filepath.Join(root, "home")
	os.MkdirAll(configsDir, 0755)
	os.MkdirAll(homeDir, 0755)

	mgr := New("")
	// Should not error, just skip missing files
	if err := mgr.Apply(configsDir, homeDir, false); err != nil {
		t.Fatalf("Apply() with missing sources returned error: %v", err)
	}
}

// contains is a helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

// containsStr is a simple strings.Contains implementation to avoid importing "strings".
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
