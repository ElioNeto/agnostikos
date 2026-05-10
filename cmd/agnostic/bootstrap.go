// Package agnostic implements the CLI commands for the AgnosticOS build system.
package agnostic

import (
	"fmt"
	"os"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	bootstrapTarget        string
	bootstrapDevice        string
	bootstrapEFIPartition  string
	bootstrapKernelVer     string
	bootstrapBusyboxVer    string
	bootstrapArch          string
	bootstrapUEFI          bool
	bootstrapJobs          string
	bootstrapSkipToolchain  bool
	bootstrapSkipKernel    bool
	bootstrapSkipBusybox   bool
	bootstrapSkipInitramfs bool
	bootstrapSkipGRUB      bool
	bootstrapForce         bool
	bootstrapDotfilesApply   bool
	bootstrapDotfilesSource  string
	bootstrapConfigsDir      string
	bootstrapAutologinUser   string
	bootstrapRecipe          string
)

var bootstrapCmd = &cobra.Command{
	Use:    "bootstrap",
	Hidden: true,
	Short:  "Create the AgnosticOS root filesystem, kernel, busybox, initramfs and GRUB (internal)",
	Long: `Build a complete bootable RootFS with:
  - FHS directory structure
  - Linux kernel (auto-downloaded generic distro kernel, compatible with all CPUs)
  - Busybox compilation (statically linked)
  - Initramfs generation
  - GRUB bootloader installation (BIOS or UEFI)

O target directory defaults to $AGNOSTICOS_ROOT ou /mnt/data/agnostikOS/rootfs.
Todo o build fica isolado em /mnt/data/agnostikOS — nada toca o sistema host.

Cada step é automático: se o artefato já existir, o step é ignorado.
Use --force para recompilar tudo mesmo que já exista.

Use --recipe <file> to load settings from a YAML recipe (kernel version, arch, UEFI).
This provides a simpler build path — the canonical way to generate bootable ISOs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := bootstrapTarget
		if len(args) > 0 {
			target = args[0]
		}

		// If --recipe was provided, load it early and apply its values as defaults.
		// Explicit CLI flags still take precedence (checked via cmd.Flags().Changed).
		if bootstrapRecipe != "" {
			data, err := os.ReadFile(bootstrapRecipe)
			if err != nil {
				return fmt.Errorf("failed to read recipe %q: %w", bootstrapRecipe, err)
			}
			var r Recipe
			if err := yaml.Unmarshal(data, &r); err != nil {
				return fmt.Errorf("failed to parse recipe %q: %w", bootstrapRecipe, err)
			}
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
		}

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

		fmt.Printf("Starting bootstrap with config: %+v\n", cfg)
		return bootstrap.BootstrapAll(cmd.Context(), cfg)
	},
}

func init() {
	bootstrapCmd.Flags().StringVarP(&bootstrapTarget, "target", "t", "", "Target directory for the rootfs (default: $AGNOSTICOS_ROOT or /mnt/data/agnostikOS/rootfs)")
	bootstrapCmd.Flags().StringVar(&bootstrapDevice, "device", "", "Disk device for BIOS grub-install (e.g. /dev/sda). Required when --uefi is not set.")
	bootstrapCmd.Flags().StringVar(&bootstrapEFIPartition, "efi-partition", "", "EFI System Partition to mount before grub-install (e.g. /dev/nvme0n1p1). Required for --uefi on real hardware.")
	bootstrapCmd.Flags().StringVar(&bootstrapKernelVer, "kernel-version", "generic", "Kernel version: 'generic' (default) = auto-detect from distro package, or specify version like '6.6'")
	bootstrapCmd.Flags().StringVar(&bootstrapBusyboxVer, "busybox-version", "1.36.1", "Busybox version (e.g. 1.36.1)")
	bootstrapCmd.Flags().StringVar(&bootstrapArch, "arch", "", "Target architecture (amd64, arm64). Empty = auto-detect from host")
	bootstrapCmd.Flags().BoolVar(&bootstrapUEFI, "uefi", false, "Enable UEFI boot support")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipToolchain, "skip-toolchain", false, "Skip toolchain compilation (binutils, gcc, glibc)")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipKernel, "skip-kernel", false, "Skip kernel installation (uses distro generic kernel)")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipBusybox, "skip-busybox", false, "Skip busybox compilation")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipInitramfs, "skip-initramfs", false, "Skip initramfs generation")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipGRUB, "skip-grub", false, "Skip GRUB installation")
	bootstrapCmd.Flags().StringVar(&bootstrapJobs, "jobs", "", "Number of parallel make jobs for toolchain (default: min(host CPUs, 4))")
	bootstrapCmd.Flags().BoolVar(&bootstrapForce, "force", false, "Force rebuild of all steps, ignoring cache")
	bootstrapCmd.Flags().BoolVar(&bootstrapDotfilesApply, "dotfiles-apply", false, "Apply dotfiles to rootfs home directory at the end of bootstrap")
	bootstrapCmd.Flags().StringVar(&bootstrapDotfilesSource, "dotfiles-source", "", "Git URL or local path for external dotfiles repository")
	bootstrapCmd.Flags().StringVar(&bootstrapConfigsDir, "configs-dir", "", "Path to the configs/ directory with embedded dotfiles")
	bootstrapCmd.Flags().StringVar(&bootstrapAutologinUser, "autologin-user", "", "Username for automatic login on tty1 (getty autologin)")
	bootstrapCmd.Flags().StringVar(&bootstrapRecipe, "recipe", "", "Path to a YAML recipe file to load defaults (kernel version, arch, UEFI). This is the canonical way to configure a bootstrap build.")
	rootCmd.AddCommand(bootstrapCmd)
}
