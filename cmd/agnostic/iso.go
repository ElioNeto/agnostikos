package agnostic

import (
	"fmt"
	"os"

	"github.com/ElioNeto/agnostikos/internal/iso"
	"github.com/spf13/cobra"
)

var (
	isoOutput string
	isoTarget string
)

var isoBuildCmd = &cobra.Command{
	Use:   "build [rootfs]",
	Short: "Build a bootable ISO from a RootFS directory",
	Long: `Build a bootable ISO image from a prepared RootFS directory.

The RootFS should contain:
  - isolinux/isolinux.bin for BIOS boot
  - boot/grub/efi.img for UEFI boot (optional)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootfs := isoTarget
		if len(args) > 0 {
			rootfs = args[0]
		}
		if rootfs == "" {
			if lfs := os.Getenv("LFS"); lfs != "" {
				rootfs = lfs
			} else {
				return fmt.Errorf("rootfs path required (provide as argument or set --target flag)")
			}
		}

		builder := iso.NewISOBuilder(rootfs, isoOutput)
		fmt.Printf("Building ISO from %s -> %s\n", rootfs, isoOutput)
		if err := builder.Build(cmd.Context(), rootfs, isoOutput); err != nil {
			return fmt.Errorf("ISO build failed: %w", err)
		}
		fmt.Printf("ISO created: %s\n", isoOutput)
		return nil
	},
}

var isoCmd = &cobra.Command{
	Use:   "iso",
	Short: "ISO image management commands",
}

func init() {
	isoBuildCmd.Flags().StringVarP(&isoOutput, "output", "o", "agnostikos.iso", "Output ISO file path")
	isoBuildCmd.Flags().StringVarP(&isoTarget, "target", "t", "", "RootFS directory (default: $LFS or /mnt/lfs)")
	isoCmd.AddCommand(isoBuildCmd)
	rootCmd.AddCommand(isoCmd)
}
