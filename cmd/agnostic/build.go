package agnostic

import (
	"fmt"
	"os"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Recipe represents a YAML recipe file for building AgnosticOS images.
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
		Arch          string `yaml:"arch"`
	} `yaml:"build"`
}

var (
	buildOutput string
	buildTarget string
)

// recipeFromPath loads a YAML recipe file and applies its values to the bootstrap
// flags as defaults. Explicit CLI flags take precedence (via cmd.Flags().Changed).
func recipeFromPath(cmd *cobra.Command, recipePath string) (name, version, outputISO string, err error) {
	data, err := os.ReadFile(recipePath)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read recipe: %w", err)
	}
	var r Recipe
	if err := yaml.Unmarshal(data, &r); err != nil {
		return "", "", "", fmt.Errorf("failed to parse recipe: %w", err)
	}
	fmt.Printf("📋 Loaded recipe: %s v%s (%s)\n", r.Name, r.Version, r.Arch)

	if !cmd.Flags().Changed("kernel-version") && r.Build.KernelVersion != "" {
		bootstrapKernelVer = r.Build.KernelVersion
	}
	if !cmd.Flags().Changed("arch") {
		recipeArch := r.Build.Arch
		if recipeArch == "" {
			recipeArch = r.Arch
		}
		if recipeArch != "" {
			bootstrapArch = recipeArch
		}
	}
	if !cmd.Flags().Changed("uefi") {
		bootstrapUEFI = r.Build.UEFI
	}

	return r.Name, r.Version, r.Build.OutputISO, nil
}

var buildCmd = &cobra.Command{
	Use:   "build [recipe.yaml]",
	Short: "Build a bootable AgnosticOS ISO (full pipeline)",
	Long: `Build a complete bootable ISO with the AgnosticOS Linux distribution.

The build command executes the full bootstrap pipeline to create a bootable ISO:
  1. Create RootFS with FHS directory structure
  2. Download and build toolchain (binutils, GCC, glibc)
  3. Compile Linux kernel
  4. Compile Busybox (statically linked)
  5. Generate initramfs
  6. Install GRUB bootloader (BIOS or UEFI)
  7. Generate bootable ISO image

If a recipe.yaml file is provided as an argument (or via --recipe), its settings
(kernel version, arch, UEFI) are loaded as defaults. Explicit CLI flags always
take precedence over recipe values.

Use --skip-toolchain to skip the lengthy toolchain compilation for faster builds
when the toolchain is already cached. See individual flags below.

Examples:
  agnostic build                                            # defaults
  agnostic build recipes/base.yaml                          # from recipe
  agnostic build --kernel-version 6.6 --uefi                # custom kernel
  agnostic build --skip-toolchain --skip-kernel             # skip expensive steps
  agnostic build recipes/base.yaml --output custom.iso      # custom output`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve target directory
		target := buildTarget
		if target == "" {
			if root := os.Getenv("AGNOSTICOS_ROOT"); root != "" {
				target = root
			}
		}

		// Handle recipe from positional argument or --recipe flag
		var recipeName, recipeVersion, recipeOutputISO string
		if len(args) > 0 {
			// Positional argument takes precedence
			n, v, o, err := recipeFromPath(cmd, args[0])
			if err != nil {
				return err
			}
			recipeName, recipeVersion, recipeOutputISO = n, v, o
		} else if bootstrapRecipe != "" {
			n, v, o, err := recipeFromPath(cmd, bootstrapRecipe)
			if err != nil {
				return err
			}
			recipeName, recipeVersion, recipeOutputISO = n, v, o
		}

		// Build BootstrapConfig from flags
		cfg := bootstrap.BootstrapConfig{
			TargetDir:      target,
			Device:         bootstrapDevice,
			EFIPartition:   bootstrapEFIPartition,
			KernelVersion:  bootstrapKernelVer,
			BusyboxVersion: bootstrapBusyboxVer,
			Arch:           bootstrapArch,
			UEFI:           bootstrapUEFI,
			Jobs:           bootstrapJobs,
			SkipToolchain:  bootstrapSkipToolchain,
			SkipKernel:     bootstrapSkipKernel,
			SkipBusybox:    bootstrapSkipBusybox,
			SkipInitramfs:  bootstrapSkipInitramfs,
			SkipGRUB:       bootstrapSkipGRUB,
			Force:          bootstrapForce,
			DotfilesApply:  bootstrapDotfilesApply,
			DotfilesSource: bootstrapDotfilesSource,
			ConfigsDir:     bootstrapConfigsDir,
			AutoLoginUser:  bootstrapAutologinUser,
		}

		// Stage 1: Full bootstrap pipeline
		fmt.Println("🚀 Starting full bootstrap pipeline...")
		if err := bootstrap.BootstrapAll(cmd.Context(), cfg); err != nil {
			return fmt.Errorf("bootstrap failed: %w", err)
		}

		// Stage 2: Generate ISO from the bootable rootfs
		resolvedTarget := target
		if resolvedTarget == "" {
			resolvedTarget = bootstrap.DefaultRoot
		}

		// Resolve ISO output path: --output flag > recipe > default
		isoOut := buildOutput
		if isoOut == "" && recipeOutputISO != "" {
			isoOut = recipeOutputISO
		}
		if isoOut == "" {
			isoOut = bootstrap.BaseDir + "/build/agnostikos-latest.iso"
		}

		// Resolve name/version from recipe or defaults
		name := recipeName
		if name == "" {
			name = "AgnostikOS"
		}
		version := recipeVersion
		if version == "" {
			version = "0.1.0"
		}

		isoCfg := bootstrap.ISOConfig{
			Name:          name,
			Version:       version,
			KernelVersion: bootstrapKernelVer,
			RootFS:        resolvedTarget,
			Output:        isoOut,
			UEFI:          bootstrapUEFI,
			BootLabel:     name + " " + version,
		}

		fmt.Printf("📀 Generating ISO from %s -> %s\n", resolvedTarget, isoOut)
		if err := bootstrap.GenerateISO(isoCfg); err != nil {
			return fmt.Errorf("ISO generation failed: %w", err)
		}

		fmt.Printf("✅ Build complete: %s\n", isoOut)
		return nil
	},
}

func init() {
	// Build-specific flags
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Output ISO path (default: /mnt/data/agnostikOS/build/agnostikos-latest.iso)")
	buildCmd.Flags().StringVarP(&buildTarget, "target", "t", "", "RootFS target directory (default: $AGNOSTICOS_ROOT or /mnt/data/agnostikOS/rootfs)")

	// Bootstrap pipeline flags (shared with bootstrap.go variables)
	buildCmd.Flags().StringVar(&bootstrapDevice, "device", "", "Disk device for BIOS grub-install (e.g. /dev/sda)")
	buildCmd.Flags().StringVar(&bootstrapEFIPartition, "efi-partition", "", "EFI System Partition for UEFI grub-install")
	buildCmd.Flags().StringVar(&bootstrapKernelVer, "kernel-version", "6.6", "Linux kernel version (e.g. 6.6)")
	buildCmd.Flags().StringVar(&bootstrapBusyboxVer, "busybox-version", "1.36.1", "Busybox version (e.g. 1.36.1)")
	buildCmd.Flags().StringVar(&bootstrapArch, "arch", "", "Target architecture (amd64, arm64). Empty = auto-detect from host")
	buildCmd.Flags().BoolVar(&bootstrapUEFI, "uefi", false, "Enable UEFI boot support")
	buildCmd.Flags().BoolVar(&bootstrapSkipToolchain, "skip-toolchain", false, "Skip toolchain compilation (binutils, gcc, glibc)")
	buildCmd.Flags().BoolVar(&bootstrapSkipKernel, "skip-kernel", false, "Skip kernel compilation")
	buildCmd.Flags().BoolVar(&bootstrapSkipBusybox, "skip-busybox", false, "Skip busybox compilation")
	buildCmd.Flags().BoolVar(&bootstrapSkipInitramfs, "skip-initramfs", false, "Skip initramfs generation")
	buildCmd.Flags().BoolVar(&bootstrapSkipGRUB, "skip-grub", false, "Skip GRUB installation")
	buildCmd.Flags().StringVar(&bootstrapJobs, "jobs", "", "Number of parallel make jobs (default: min(CPUs, 4))")
	buildCmd.Flags().BoolVar(&bootstrapForce, "force", false, "Force rebuild of all steps, ignoring cache")
	buildCmd.Flags().BoolVar(&bootstrapDotfilesApply, "dotfiles-apply", false, "Apply dotfiles to rootfs home directory at the end")
	buildCmd.Flags().StringVar(&bootstrapDotfilesSource, "dotfiles-source", "", "Git URL or local path for external dotfiles")
	buildCmd.Flags().StringVar(&bootstrapConfigsDir, "configs-dir", "", "Path to the configs/ directory with embedded dotfiles")
	buildCmd.Flags().StringVar(&bootstrapAutologinUser, "autologin-user", "", "Username for automatic login on tty1 (getty autologin)")
	buildCmd.Flags().StringVar(&bootstrapRecipe, "recipe", "", "Path to a YAML recipe file to load defaults (kernel version, arch, UEFI)")

	rootCmd.AddCommand(buildCmd)
}
