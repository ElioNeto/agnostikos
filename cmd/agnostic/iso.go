package agnostic

import (
	"fmt"

	"github.com/ElioNeto/agnostikos/internal/bootstrap"
	"github.com/spf13/cobra"
)

var (
	isoRootFS  string
	isoOutput  string
	isoVersion string
	isoName    string
	isoUEFI    bool
)

var isoCmd = &cobra.Command{
	Use:   "iso",
	Short: "Generate a bootable ISO from the RootFS",
	Long: `Generate a bootable ISO image from an existing RootFS.

The RootFS must already contain boot/vmlinuz-<version> and boot/initramfs.img.
Run 'agnostic bootstrap' first to build those artifacts.

Examples:
  agnostic iso
  agnostic iso --rootfs /mnt/data/agnostikOS/rootfs --output /mnt/data/agnostikOS/build/agnostikos.iso
  agnostic iso --uefi`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isoRootFS == "" {
			if v := bootstrap.DefaultRoot; v != "" {
				isoRootFS = v
			}
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
			Name:      isoName,
			Version:   isoVersion,
			RootFS:    isoRootFS,
			Output:    isoOutput,
			UEFI:      isoUEFI,
			BootLabel: isoName + "-" + isoVersion,
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
	isoCmd.Flags().BoolVar(&isoUEFI, "uefi", false, "Generate UEFI-bootable ISO")
	rootCmd.AddCommand(isoCmd)
}
