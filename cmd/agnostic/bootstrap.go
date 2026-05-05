package agnostic

import (
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
)

var (
	bootstrapTarget        string
	bootstrapDevice        string
	bootstrapEFIPartition  string
	bootstrapKernelVer     string
	bootstrapBusyboxVer    string
	bootstrapUEFI          bool
	bootstrapSkipKernel    bool
	bootstrapSkipBusybox   bool
	bootstrapSkipInitramfs bool
	bootstrapSkipGRUB      bool
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Create the AgnosticOS root filesystem, kernel, busybox, initramfs and GRUB",
	Long: `Build a complete bootable RootFS with:
  - FHS directory structure
  - Linux kernel compilation
  - Busybox compilation (statically linked)
  - Initramfs generation
  - GRUB bootloader installation (BIOS or UEFI)

The target directory defaults to $AGNOSTICOS_ROOT or /mnt/agnosticOS.
Build artifacts (toolchain sources) are kept outside the rootfs at <target>/../sources.

Flags allow skipping individual steps and enabling UEFI support.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := bootstrapTarget
		if len(args) > 0 {
			target = args[0]
		}

		cfg := bootstrap.BootstrapConfig{
			TargetDir:      target,
			Device:         bootstrapDevice,
			EFIPartition:   bootstrapEFIPartition,
			KernelVersion:  bootstrapKernelVer,
			BusyboxVersion: bootstrapBusyboxVer,
			UEFI:           bootstrapUEFI,
			SkipKernel:     bootstrapSkipKernel,
			SkipBusybox:    bootstrapSkipBusybox,
			SkipInitramfs:  bootstrapSkipInitramfs,
			SkipGRUB:       bootstrapSkipGRUB,
		}

		fmt.Printf("Starting bootstrap with config: %+v\n", cfg)
		return bootstrap.BootstrapAll(cmd.Context(), cfg)
	},
}

func init() {
	bootstrapCmd.Flags().StringVarP(&bootstrapTarget, "target", "t", "", "Target directory for the rootfs (default: $AGNOSTICOS_ROOT or /mnt/agnosticOS)")
	bootstrapCmd.Flags().StringVar(&bootstrapDevice, "device", "", "Disk device for BIOS grub-install (e.g. /dev/sda). Required when --uefi is not set.")
	bootstrapCmd.Flags().StringVar(&bootstrapEFIPartition, "efi-partition", "", "EFI System Partition to mount before grub-install (e.g. /dev/nvme0n1p1). Required for --uefi on real hardware.")
	bootstrapCmd.Flags().StringVar(&bootstrapKernelVer, "kernel-version", "6.6", "Linux kernel version (e.g. 6.6)")
	bootstrapCmd.Flags().StringVar(&bootstrapBusyboxVer, "busybox-version", "1.36.1", "Busybox version (e.g. 1.36.1)")
	bootstrapCmd.Flags().BoolVar(&bootstrapUEFI, "uefi", false, "Enable UEFI boot support")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipKernel, "skip-kernel", false, "Skip kernel compilation")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipBusybox, "skip-busybox", false, "Skip busybox compilation")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipInitramfs, "skip-initramfs", false, "Skip initramfs generation")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipGRUB, "skip-grub", false, "Skip GRUB installation")
	rootCmd.AddCommand(bootstrapCmd)
}
