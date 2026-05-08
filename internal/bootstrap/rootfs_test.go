package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTarget_Arg(t *testing.T) {
	if got := resolveTarget("/custom"); got != "/custom" {
		t.Errorf("expected /custom, got %s", got)
	}
}

func TestResolveTarget_EnvVar(t *testing.T) {
	t.Setenv("AGNOSTICOS_ROOT", "/from-env")
	if got := resolveTarget(""); got != "/from-env" {
		t.Errorf("expected /from-env, got %s", got)
	}
}

func TestResolveTarget_Default(t *testing.T) {
	_ = os.Unsetenv("AGNOSTICOS_ROOT")
	if got := resolveTarget(""); got != DefaultRoot {
		t.Errorf("expected %s, got %s", DefaultRoot, got)
	}
}

func TestFHSDirectories_Created(t *testing.T) {
	tmp, err := os.MkdirTemp("", "lfs-fhs-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	for _, dir := range FHSDirectories {
		path := filepath.Join(tmp, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Errorf("failed to create dir %s: %v", dir, err)
		}
	}

	for _, dir := range FHSDirectories {
		path := filepath.Join(tmp, dir)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("directory missing: %s", dir)
		}
	}
}

func TestFHSDirectories_Count(t *testing.T) {
	if len(FHSDirectories) < 23 {
		t.Errorf("expected at least 23 FHS directories, got %d", len(FHSDirectories))
	}
}

func TestDownloadToolchain_SkipsExisting(t *testing.T) {
	tmp, err := os.MkdirTemp("", "lfs-toolchain-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	// Estrutura de diretórios real: rootfs/ e sources/ ficam no mesmo base dir
	rootfsDir := filepath.Join(tmp, "rootfs")
	sourcesDir := filepath.Join(tmp, "sources")
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		t.Fatalf("failed to create sourcesDir: %v", err)
	}

	// cria arquivos falsos simulando downloads já feitos
	for _, pkg := range DefaultToolchain {
		dest := filepath.Join(sourcesDir, filepath.Base(pkg.URL))
		if err := os.WriteFile(dest, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to write fake file %s: %v", dest, err)
		}
	}

	// como todos já existem, não deve tentar baixar nada (sem rede)
	if err := DownloadToolchain(rootfsDir); err != nil {
		t.Errorf("expected no error when files exist, got: %v", err)
	}
}

func TestConfigureDefaultShell(t *testing.T) {
	tests := []struct {
		name       string
		prepShells string // initial /etc/shells content ("" means file does not exist)
		prepPasswd string // initial /etc/passwd content ("" means file does not exist)
		hasZsh     bool   // whether /bin/zsh exists in rootfs
		wantShells string // expected /etc/shells content after call
		wantPasswd string // expected /etc/passwd content after call
	}{
		{
			name:       "happy path adds zsh to both shells and passwd",
			prepShells: "",
			prepPasswd: "root:x:0:0:root:/root:/bin/sh\n",
			hasZsh:     true,
			wantShells: "/bin/zsh\n",
			wantPasswd: "root:x:0:0:root:/root:/bin/zsh\n",
		},
		{
			name:       "zsh not found skips gracefully",
			prepShells: "",
			prepPasswd: "",
			hasZsh:     false,
			wantShells: "",
			wantPasswd: "",
		},
		{
			name:       "shells already contains /bin/zsh is idempotent",
			prepShells: "/bin/bash\n/bin/zsh\n",
			prepPasswd: "root:x:0:0:root:/root:/bin/sh\n",
			hasZsh:     true,
			wantShells: "/bin/bash\n/bin/zsh\n",
			wantPasswd: "root:x:0:0:root:/root:/bin/zsh\n",
		},
		{
			name:       "passwd already has root shell set to /bin/zsh is idempotent",
			prepShells: "",
			prepPasswd: "root:x:0:0:root:/root:/bin/zsh\n",
			hasZsh:     true,
			wantShells: "/bin/zsh\n",
			wantPasswd: "root:x:0:0:root:/root:/bin/zsh\n",
		},
		{
			name:       "shells file does not exist yet creates it",
			prepShells: "",
			prepPasswd: "root:x:0:0:root:/root:/bin/sh\n",
			hasZsh:     true,
			wantShells: "/bin/zsh\n",
			wantPasswd: "root:x:0:0:root:/root:/bin/zsh\n",
		},
		{
			name:       "passwd file does not exist yet creates it",
			prepShells: "",
			prepPasswd: "",
			hasZsh:     true,
			wantShells: "/bin/zsh\n",
			wantPasswd: "\nroot:x:0:0:root:/root:/bin/zsh",
		},
		{
			name:       "both shells and passwd already fully configured",
			prepShells: "/bin/zsh\n",
			prepPasswd: "root:x:0:0:root:/root:/bin/zsh\n",
			hasZsh:     true,
			wantShells: "/bin/zsh\n",
			wantPasswd: "root:x:0:0:root:/root:/bin/zsh\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()

			// Conditionally create /bin/zsh
			if tt.hasZsh {
				zshPath := filepath.Join(tmp, "bin", "zsh")
				if err := os.MkdirAll(filepath.Dir(zshPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(zshPath, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
					t.Fatal(err)
				}
			}

			// Conditionally create /etc/shells
			if tt.prepShells != "" {
				shellsPath := filepath.Join(tmp, "etc", "shells")
				if err := os.MkdirAll(filepath.Dir(shellsPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(shellsPath, []byte(tt.prepShells), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Conditionally create /etc/passwd
			if tt.prepPasswd != "" {
				passwdPath := filepath.Join(tmp, "etc", "passwd")
				if err := os.MkdirAll(filepath.Dir(passwdPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(passwdPath, []byte(tt.prepPasswd), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Execute
			err := configureDefaultShell(tmp)
			if err != nil {
				t.Fatalf("configureDefaultShell() returned unexpected error: %v", err)
			}

			// Assert /etc/shells
			shellsPath := filepath.Join(tmp, "etc", "shells")
			if tt.wantShells == "" {
				if _, statErr := os.Stat(shellsPath); !os.IsNotExist(statErr) {
					t.Errorf("expected /etc/shells to not exist, but it does")
				}
			} else {
				data, readErr := os.ReadFile(shellsPath)
				if readErr != nil {
					t.Fatalf("failed to read /etc/shells: %v", readErr)
				}
				if string(data) != tt.wantShells {
					t.Errorf("/etc/shells content mismatch:\ngot:  %q\nwant: %q", string(data), tt.wantShells)
				}
			}

			// Assert /etc/passwd
			passwdPath := filepath.Join(tmp, "etc", "passwd")
			if tt.wantPasswd == "" {
				if _, statErr := os.Stat(passwdPath); !os.IsNotExist(statErr) {
					t.Errorf("expected /etc/passwd to not exist, but it does")
				}
			} else {
				data, readErr := os.ReadFile(passwdPath)
				if readErr != nil {
					t.Fatalf("failed to read /etc/passwd: %v", readErr)
				}
				if string(data) != tt.wantPasswd {
					t.Errorf("/etc/passwd content mismatch:\ngot:  %q\nwant: %q", string(data), tt.wantPasswd)
				}
			}
		})
	}
}

func TestDefaultToolchain_HasRequiredPackages(t *testing.T) {
	required := []string{"binutils", "gcc", "glibc"}
	for _, req := range required {
		found := false
		for _, pkg := range DefaultToolchain {
			if len(pkg.Name) >= len(req) && pkg.Name[:len(req)] == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected toolchain to contain %s", req)
		}
	}
}
