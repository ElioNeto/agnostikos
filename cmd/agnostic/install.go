package agnostic

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallCmd(t *testing.T) {
	t.Run("install package via pacman", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "pacman"
		installCmd.Execute()
		assert.Contains(t, output.String(), "📦 Installing 'test-package' via pacman...")
	})

	t.Run("install package via nix", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "nix"
		installCmd.Execute()
		assert.Contains(t, output.String(), "📦 Installing 'test-package' via nix...")
	})

	t.Run("install package via flatpak", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "flatpak"
		installCmd.Execute()
		assert.Contains(t, output.String(), "📦 Installing 'test-package' via flatpak...")
	})

	t.Run("install package in isolated namespace", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "pacman"
		isolated = true
		installCmd.Execute()
		assert.Contains(t, output.String(), "📦 Installing 'test-package' via pacman...")
		assert.Contains(t, output.String(), "🔒 Running in isolated namespace...")
	})
}

func TestRemoveCmd(t *testing.T) {
	t.Run("remove package via pacman", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "pacman"
		removeCmd.Execute()
		assert.Contains(t, output.String(), "🗑️  Removing 'test-package' via pacman...")
	})

	t.Run("remove package via nix", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "nix"
		removeCmd.Execute()
		assert.Contains(t, output.String(), "🗑️  Removing 'test-package' via nix...")
	})

	t.Run("remove package via flatpak", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "flatpak"
		removeCmd.Execute()
		assert.Contains(t, output.String(), "🗑️  Removing 'test-package' via flatpak...")
	})
}

func TestUpdateCmd(t *testing.T) {
	t.Run("update packages via pacman", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "pacman"
		updateCmd.Execute()
		assert.Contains(t, output.String(), "🔄 Updating via pacman...")
	})

	t.Run("update packages via nix", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "nix"
		updateCmd.Execute()
		assert.Contains(t, output.String(), "🔄 Updating via nix...")
	})

	t.Run("update packages via flatpak", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "flatpak"
		updateCmd.Execute()
		assert.Contains(t, output.String(), "🔄 Updating via flatpak...")
	})
}

func TestSearchCmd(t *testing.T) {
	t.Run("search package via pacman", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "pacman"
		searchCmd.Execute()
		assert.Contains(t, output.String(), "🔍 Searching 'test-query' in pacman...")
	})

	t.Run("search package via nix", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "nix"
		searchCmd.Execute()
		assert.Contains(t, output.String(), "🔍 Searching 'test-query' in nix...")
	})

	t.Run("search package via flatpak", func(t *testing.T) {
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		backend = "flatpak"
		searchCmd.Execute()
		assert.Contains(t, output.String(), "🔍 Searching 'test-query' in flatpak...")
	})
}