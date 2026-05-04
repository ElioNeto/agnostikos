package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const DefaultLFSRoot = "/mnt/lfs"

// FHSDirectories é a árvore de diretórios do Filesystem Hierarchy Standard
var FHSDirectories = []string{
	"bin", "boot", "dev", "etc", "home", "lib", "lib64",
	"media", "mnt", "opt", "proc", "root", "run", "sbin",
	"srv", "sys", "tmp",
	"usr/bin", "usr/lib", "usr/sbin", "usr/include", "usr/share", "usr/local", "usr/src",
	"var/cache", "var/lib", "var/log", "var/run", "var/tmp",
	"tools", "sources",
}

// CreateRootFS monta a árvore FHS no diretório alvo e inicializa o VirtualFS
func CreateRootFS(target string) error {
	if target == "" {
		if lfs := os.Getenv("LFS"); lfs != "" {
			target = lfs
		} else {
			target = DefaultLFSRoot
		}
	}
	fmt.Printf("[rootfs] Creating RootFS at: %s\n", target)

	for _, dir := range FHSDirectories {
		path := filepath.Join(target, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}
	}

	// Symlinks modernos (usrmerge)
	symlinks := map[string]string{
		filepath.Join(target, "lib"):     "usr/lib",
		filepath.Join(target, "lib64"):   "usr/lib",
		filepath.Join(target, "bin"):     "usr/bin",
		filepath.Join(target, "sbin"):    "usr/sbin",
		filepath.Join(target, "var/run"): "../run",
	}
	for link, dest := range symlinks {
		os.Remove(link)
		if err := os.Symlink(dest, link); err != nil {
			fmt.Printf("[rootfs] warn: symlink %s -> %s: %v\n", link, dest, err)
		}
	}

	fmt.Println("[rootfs] FHS structure created")
	return mountVirtualFS(target)
}

// mountVirtualFS monta proc/sys/dev dentro do chroot
func mountVirtualFS(target string) error {
	type mountSpec struct {
		fstype, source, target, opts string
	}
	mounts := []mountSpec{
		{"proc", "proc", filepath.Join(target, "proc"), ""},
		{"sysfs", "sysfs", filepath.Join(target, "sys"), ""},
		{"devtmpfs", "devtmpfs", filepath.Join(target, "dev"), "mode=0755"},
		{"devpts", "devpts", filepath.Join(target, "dev/pts"), "gid=5,mode=0620"},
		{"tmpfs", "tmpfs", filepath.Join(target, "run"), ""},
	}
	for _, m := range mounts {
		args := []string{"-t", m.fstype}
		if m.opts != "" {
			args = append(args, "-o", m.opts)
		}
		args = append(args, m.source, m.target)
		if out, err := exec.Command("mount", args...).CombinedOutput(); err != nil {
			fmt.Printf("[rootfs] warn: mount %s: %s\n", m.fstype, string(out))
		} else {
			fmt.Printf("[rootfs] mounted %s -> %s\n", m.fstype, m.target)
		}
	}
	return nil
}

// UnmountVirtualFS desmonta os filesystems virtuais do chroot
func UnmountVirtualFS(target string) error {
	for _, p := range []string{"dev/pts", "dev", "run", "proc", "sys"} {
		exec.Command("umount", filepath.Join(target, p)).Run()
	}
	return nil
}

func DownloadToolchain(sourcesDir string) error {
	packages := []struct{ name, url string }{
		{"binutils-2.42", "https://sourceware.org/pub/binutils/releases/binutils-2.42.tar.xz"},
		{"gcc-14.1.0", "https://ftp.gnu.org/gnu/gcc/gcc-14.1.0/gcc-14.1.0.tar.xz"},
		{"glibc-2.39", "https://ftp.gnu.org/gnu/glibc/glibc-2.39.tar.xz"},
	}
	for _, pkg := range packages {
		dest := filepath.Join(sourcesDir, filepath.Base(pkg.url))
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("[toolchain] already exists: %s\n", pkg.name)
			continue
		}
		if err := downloadFile(dest, pkg.url); err != nil {
			return fmt.Errorf("download %s: %w", pkg.name, err)
		}
	}
	return nil
}
