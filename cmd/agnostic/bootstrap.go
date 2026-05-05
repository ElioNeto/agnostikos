package agnostic

import (
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
)

var (
	bootstrapTarget        string
	bootstrapKernelVer     string
	bootstrapBusyboxVer    string
	bootstrapUEFI          bool
	bootstrapSkipKernel    bool
	bootstrapSkipBusybox   bool
	bootstrapSkipInitramfs bool
	bootstrapSkipGRUB      bool
	bootstrapDevice        string
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Create the LFS root filesystem structure with kernel, busybox, initramfs and GRUB",
	Long: `Build a complete bootable RootFS with:
  - FHS directory structure
  - Linux kernel compilation
  - Busybox compilation
  - Initramfs generation
  - GRUB bootloader installation

Flags allow skipping individual steps and enabling UEFI support.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := bootstrapTarget
		if len(args) > 0 {
			target = args[0]
		}

		cfg := bootstrap.BootstrapConfig{
			TargetDir:      target,
			KernelVersion:  bootstrapKernelVer,
			BusyboxVersion: bootstrapBusyboxVer,
			UEFI:           bootstrapUEFI,
			SkipKernel:     bootstrapSkipKernel,
			SkipBusybox:    bootstrapSkipBusybox,
			SkipInitramfs:  bootstrapSkipInitramfs,
			SkipGRUB:       bootstrapSkipGRUB,
			Device:         bootstrapDevice,
		}

		fmt.Printf("Starting bootstrap with config: %+v\n", cfg)
		return bootstrap.BootstrapAll(cmd.Context(), cfg)
	},
}

func init() {
	bootstrapCmd.Flags().StringVarP(&bootstrapTarget, "target", "t", "", "Target directory (default: $LFS or /mnt/lfs)")
	bootstrapCmd.Flags().StringVar(&bootstrapKernelVer, "kernel-version", "6.6", "Linux kernel version (e.g. 6.6)")
	bootstrapCmd.Flags().StringVar(&bootstrapBusyboxVer, "busybox-version", "1.36.1", "Busybox version (e.g. 1.36.1)")
	bootstrapCmd.Flags().BoolVar(&bootstrapUEFI, "uefi", false, "Enable UEFI boot support")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipKernel, "skip-kernel", false, "Skip kernel compilation")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipBusybox, "skip-busybox", false, "Skip busybox compilation")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipInitramfs, "skip-initramfs", false, "Skip initramfs generation")
	bootstrapCmd.Flags().BoolVar(&bootstrapSkipGRUB, "skip-grub", false, "Skip GRUB installation")
	bootstrapCmd.Flags().StringVar(&bootstrapDevice, "device", "", "Disk device for BIOS grub-install (e.g. /dev/sda). Required when --uefi is not set.")
	rootCmd.AddCommand(bootstrapCmd)
}
