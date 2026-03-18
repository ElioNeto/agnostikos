package agnostic

import (
	"fmt"
	"os"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Recipe struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Arch        string   `yaml:"arch"`
	Description string   `yaml:"description"`
	Packages    []string `yaml:"packages"`
	Build       struct {
			KernelVersion string `yaml:"kernel_version"`
			OutputISO     string `yaml:"output_iso"`
			UEFI          bool   `yaml:"uefi"`
		} `yaml:"build"`
}

var (
	buildOutput string
	buildTarget string
)

var buildCmd = &cobra.Command{
	Use:   "build [recipe.yaml]",
	Short: "Build an AgnosticOS image from a recipe",
	Long: `Build a bootable ISO from a YAML recipe.

Steps:
  1. Create RootFS with FHS structure
  2. Compile kernel (if specified)
  3. Generate bootable ISO

Example:
  agnostic build recipes/base.yaml
  agnostic build recipes/base.yaml --output custom.iso`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read recipe: %w", err)
		}
		var r Recipe
		if err := yaml.Unmarshal(data, &r); err != nil {
			return fmt.Errorf("failed to parse recipe: %w", err)
		}
		fmt.Printf("🏗️  Building %s v%s (%s)\n", r.Name, r.Version, r.Arch)

		if err := bootstrap.CreateRootFS(buildTarget); err != nil {
			return fmt.Errorf("rootfs: %w", err)
		}
		if r.Build.KernelVersion != "" {
			kCfg := bootstrap.KernelConfig{
				Version:    r.Build.KernelVersion,
				SourcesDir: buildTarget + "/sources",
				OutputDir:  buildTarget + "/boot",
				Defconfig:  "x86_64_defconfig",
			}
			if err := bootstrap.BuildKernel(kCfg); err != nil {
				return fmt.Errorf("kernel: %w", err)
			}
		}
		out := buildOutput
		if out == "" {
			out = r.Build.OutputISO
		}
		isoCfg := bootstrap.ISOConfig{
			Name:      r.Name,
			Version:   r.Version,
			RootFS:    buildTarget,
			Output:    out,
			UEFI:      r.Build.UEFI,
			BootLabel: r.Name + " " + r.Version,
		}
		if err := bootstrap.GenerateISO(isoCfg); err != nil {
			return fmt.Errorf("iso: %w", err)
		}
		fmt.Printf("✅ Build complete: %s\n", out)
		return nil
	},
}

func init() {
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Override ISO output path")
	buildCmd.Flags().StringVarP(&buildTarget, "target", "t", "/mnt/lfs", "RootFS target directory")
	rootCmd.AddCommand(buildCmd)
}