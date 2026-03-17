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
		target = DefaultLFSRoot
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
