package main

import (
	"fmt"
	"os"

	"github.com/ElioNeto/agnostikos/cmd/agnostic"
	"github.com/spf13/cobra/doc"
)

func main() {
	if err := generate(); err != nil {
		fmt.Fprintf(os.Stderr, "error generating docs: %v\n", err)
		os.Exit(1)
	}
}

func generate() error {
	rootCmd := agnostic.RootCmd()

	// Generate man pages into docs/man/ (section 1)
	manHeader := &doc.GenManHeader{
		Title:   "AGNOSTIC",
		Section: "1",
	}
	manDir := "docs/man"
	if err := os.MkdirAll(manDir, 0o755); err != nil {
		return fmt.Errorf("creating man dir: %w", err)
	}
	if err := doc.GenManTree(rootCmd, manHeader, manDir); err != nil {
		return fmt.Errorf("generating man pages: %w", err)
	}
	fmt.Printf("✅ Man pages generated in %s/\n", manDir)

	// Generate Markdown docs into docs/commands/
	mdDir := "docs/commands"
	if err := os.MkdirAll(mdDir, 0o755); err != nil {
		return fmt.Errorf("creating markdown dir: %w", err)
	}
	if err := doc.GenMarkdownTree(rootCmd, mdDir); err != nil {
		return fmt.Errorf("generating markdown docs: %w", err)
	}
	fmt.Printf("✅ Markdown docs generated in %s/\n", mdDir)

	return nil
}
