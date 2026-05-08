package agnostic

import (
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
)

var (
	isoRootFS        string
	isoOutput        string
	isoVersion       string
	isoName          string
	isoKernelVersion string
	isoInitramfs     string
	isoUEFI          bool
	isoTestMode      bool
)

var isoCmd = &cobra.Command{
	Use:   "iso",
	Short: "Generate a bootable ISO from the RootFS",
	Long: `Generate a bootable ISO image from an existing RootFS.

The RootFS must already contain boot/vmlinuz-<kernel-version> and boot/initramfs.img.
Run 'agnostic bootstrap' first to build those artifacts.

Se --kernel-version não for informado, o comando localiza automaticamente
o primeiro vmlinuz-* encontrado em boot/.

Examples:
  agnostic iso
  agnostic iso --kernel-version 6.6
  agnostic iso --rootfs /mnt/data/agnostikOS/rootfs --output /mnt/data/agnostikOS/build/agnostikos.iso
  agnostic iso --uefi`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isoRootFS == "" {
			isoRootFS = bootstrap.DefaultRoot
		}
		if isoOutput == "" {
			isoOutput = bootstrap.BaseDir + "/build/agnostikos-latest.iso"
		}
		if isoName == "" {
			isoName = "AgnostikOS"
		}
		if isoVersion == "" {
			isoVersion = "0.1.0"
		}

		fmt.Printf("Building ISO from %s -> %s\n", isoRootFS, isoOutput)

		cfg := bootstrap.ISOConfig{
			Name:          isoName,
			Version:       isoVersion,
			KernelVersion: isoKernelVersion, // vazio = auto-detect por glob
			RootFS:        isoRootFS,
			Output:        isoOutput,
			InitramfsPath: isoInitramfs,
			UEFI:          isoUEFI,
			BootLabel:     isoName + "-" + isoVersion,
			TestMode:      isoTestMode,
		}

		if err := bootstrap.GenerateISO(cfg); err != nil {
			return fmt.Errorf("ISO build failed: %w", err)
		}

		fmt.Printf("\u2705 ISO ready: %s\n", isoOutput)
		return nil
	},
}

func init() {
	isoCmd.Flags().StringVar(&isoRootFS, "rootfs", "", "RootFS directory (default: $AGNOSTICOS_ROOT or /mnt/data/agnostikOS/rootfs)")
	isoCmd.Flags().StringVarP(&isoOutput, "output", "o", "", "Output ISO path (default: /mnt/data/agnostikOS/build/agnostikos-latest.iso)")
	isoCmd.Flags().StringVar(&isoVersion, "version", "0.1.0", "OS version embedded in ISO label")
	isoCmd.Flags().StringVar(&isoName, "name", "AgnostikOS", "OS name embedded in ISO label")
	isoCmd.Flags().StringVar(&isoKernelVersion, "kernel-version", "", "Kernel version to use (ex: 6.6). If empty, auto-detects from boot/vmlinuz-*")
	isoCmd.Flags().StringVar(&isoInitramfs, "initramfs", "", "Custom initramfs path (default: RootFS/boot/initramfs.img)")
	isoCmd.Flags().BoolVar(&isoUEFI, "uefi", false, "Generate UEFI-bootable ISO")
	isoCmd.Flags().BoolVar(&isoTestMode, "test", false, "Generate test ISO with minimal initramfs (no busybox, uses host kernel only)")
	rootCmd.AddCommand(isoCmd)
}
