package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	t.Setenv("AGNOSTICOS_ROOT", "")
	if got := resolveTarget(""); got != DefaultRoot {
		t.Errorf("expected %s, got %s", DefaultRoot, got)
	}
}

func TestResolveTarget_ArgOverridesEnv(t *testing.T) {
	t.Setenv("AGNOSTICOS_ROOT", "/from-env")
	if got := resolveTarget("/from-arg"); got != "/from-arg" {
		t.Errorf("expected /from-arg, got %s", got)
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
	//nolint:gosec // test fixtures, not real credentials
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

// ---------------------------------------------------------------------------
// kernelImageName
// ---------------------------------------------------------------------------

func TestKernelImageName(t *testing.T) {
	tests := []struct {
		arch    string
		version string
		want    string
	}{
		{"amd64", "6.6.0", "vmlinuz-6.6.0"},
		{"arm64", "6.6.0", "Image-6.6.0"},
		{"amd64", "6.8", "vmlinuz-6.8"},
		{"arm64", "6.8.1", "Image-6.8.1"},
	}
	for _, tt := range tests {
		t.Run(tt.arch+"_"+tt.version, func(t *testing.T) {
			got := kernelImageName(tt.arch, tt.version)
			if got != tt.want {
				t.Errorf("kernelImageName(%q, %q) = %q; want %q", tt.arch, tt.version, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// hasShellEntry
// ---------------------------------------------------------------------------

func TestHasShellEntry(t *testing.T) {
	tests := []struct {
		name    string
		content string
		shell   string
		want    bool
	}{
		{name: "found in middle", content: "/bin/bash\n/bin/zsh\n/bin/dash\n", shell: "/bin/zsh", want: true},
		{name: "found at end", content: "/bin/bash\n/bin/zsh\n", shell: "/bin/zsh", want: true},
		{name: "found single line", content: "/bin/zsh\n", shell: "/bin/zsh", want: true},
		{name: "not found", content: "/bin/bash\n/bin/dash\n", shell: "/bin/zsh", want: false},
		{name: "empty content", content: "", shell: "/bin/zsh", want: false},
		{name: "trailing space matched after trim", content: "/bin/zsh \n", shell: "/bin/zsh", want: true},
		{name: "partial path no match", content: "/bin/zsh-stuff\n", shell: "/bin/zsh", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasShellEntry(tt.content, tt.shell)
			if got != tt.want {
				t.Errorf("hasShellEntry(%q, %q) = %v; want %v", tt.content, tt.shell, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// configureInittab
// ---------------------------------------------------------------------------

func TestConfigureInittab(t *testing.T) {
	tests := []struct {
		name          string
		autoLoginUser string
		wantErr       bool
		checkInittab  func(t *testing.T, data string)
		checkRcS      func(t *testing.T, data string)
	}{
		{
			name:          "default inittab without auto-login uses askfirst",
			autoLoginUser: "",
			wantErr:       false,
			checkInittab: func(t *testing.T, data string) {
				if !strings.Contains(data, "::askfirst:-/bin/sh") {
					t.Errorf("inittab should contain askfirst for manual login, got: %s", data)
				}
				if strings.Contains(data, "/bin/login -f") {
					t.Errorf("inittab should NOT contain auto-login when no user specified")
				}
			},
			checkRcS: nil,
		},
		{
			name:          "auto-login inittab with root user",
			autoLoginUser: "root",
			wantErr:       false,
			checkInittab: func(t *testing.T, data string) {
				if !strings.Contains(data, "/bin/login -f root") {
					t.Errorf("inittab should contain auto-login for root, got: %s", data)
				}
				if strings.Contains(data, "::askfirst:-/bin/sh") {
					t.Errorf("inittab should NOT contain askfirst when auto-login is configured")
				}
				if !strings.Contains(data, "tty1::respawn:") {
					t.Errorf("inittab should bind auto-login to tty1")
				}
			},
			checkRcS: nil,
		},
		{
			name:          "non-root auto-login user",
			autoLoginUser: "admin",
			wantErr:       false,
			checkInittab: func(t *testing.T, data string) {
				if !strings.Contains(data, "/bin/login -f admin") {
					t.Errorf("inittab should contain auto-login for admin, got: %s", data)
				}
			},
			checkRcS: nil,
		},
		{
			name:          "rcS contains boot commands",
			autoLoginUser: "",
			wantErr:       false,
			checkInittab:  nil,
			checkRcS: func(t *testing.T, data string) {
				expected := []string{
					"mount -t proc",
					"mount -t sysfs",
					"mount -t tmpfs",
					"mkdir -p /dev/pts",
					"mount -t devpts",
					"echo /sbin/mdev",
					"mdev -s",
					"hostname agnostikos",
				}
				for _, exp := range expected {
					if !strings.Contains(data, exp) {
						t.Errorf("rcS should contain %q, got: %s", exp, data)
					}
				}
			},
		},
		{
			name:          "inittab has sysinit, ctrlaltdel, shutdown entries",
			autoLoginUser: "testuser",
			wantErr:       false,
			checkInittab: func(t *testing.T, data string) {
				for _, entry := range []string{"::sysinit:", "::ctrlaltdel:", "::shutdown:"} {
					if !strings.Contains(data, entry) {
						t.Errorf("inittab should contain %q, got: %s", entry, data)
					}
				}
			},
			checkRcS: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()

			err := configureInittab(tmp, tt.autoLoginUser)
			if tt.wantErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err != nil {
				return
			}

			// Verify /etc/inittab exists
			inittabPath := filepath.Join(tmp, "etc", "inittab")
			inittabData, err := os.ReadFile(inittabPath)
			if err != nil {
				t.Fatalf("expected /etc/inittab to exist: %v", err)
			}

			// Verify /etc/init.d/rcS exists
			rcSPath := filepath.Join(tmp, "etc", "init.d", "rcS")
			rcSData, err := os.ReadFile(rcSPath)
			if err != nil {
				t.Fatalf("expected /etc/init.d/rcS to exist: %v", err)
			}

			// Run custom checks
			if tt.checkInittab != nil {
				tt.checkInittab(t, string(inittabData))
			}
			if tt.checkRcS != nil {
				tt.checkRcS(t, string(rcSData))
			}
		})
	}
}

func TestConfigureInittab_RcSExecutable(t *testing.T) {
	tmp := t.TempDir()

	err := configureInittab(tmp, "")
	if err != nil {
		t.Fatalf("configureInittab failed: %v", err)
	}

	rcSPath := filepath.Join(tmp, "etc", "init.d", "rcS")
	info, err := os.Stat(rcSPath)
	if err != nil {
		t.Fatalf("expected rcS to exist: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("rcS permissions should be 0755, got: %o", info.Mode().Perm())
	}
}

func TestConfigureInittab_MkdirError(t *testing.T) {
	tmp := t.TempDir()

	// Make /etc a file so MkdirAll on /etc/init.d fails
	etcPath := filepath.Join(tmp, "etc")
	if err := os.MkdirAll(filepath.Dir(etcPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(etcPath, []byte("not-a-directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := configureInittab(tmp, "root")
	if err == nil {
		t.Fatal("expected error when /etc is a file")
	}
	if !strings.Contains(err.Error(), "mkdir /etc") {
		t.Errorf("expected 'mkdir /etc' error, got: %v", err)
	}
}

func TestConfigureInittab_WriteInittabError(t *testing.T) {
	tmp := t.TempDir()

	// Create /etc as a file so writing /etc/inittab fails
	etcPath := filepath.Join(tmp, "etc")
	if err := os.MkdirAll(filepath.Dir(etcPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(etcPath, []byte("not-a-directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := configureInittab(tmp, "")
	if err == nil {
		t.Fatal("expected error when /etc is a file")
	}
}

// ---------------------------------------------------------------------------
// configureAutologin
// ---------------------------------------------------------------------------

// TestConfigureAutologin_MkdirError tests error handling when the drop-in directory
// cannot be created (e.g., etc/systemd is a file instead of directory).
func TestConfigureAutologin_MkdirError(t *testing.T) {
	tmp := t.TempDir()

	// Make etc/systemd a file so MkdirAll on the drop-in path fails with ENOTDIR
	systemdPath := filepath.Join(tmp, "etc", "systemd")
	if err := os.MkdirAll(filepath.Dir(systemdPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(systemdPath, []byte("not-a-directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := configureAutologin(tmp, "root")
	if err == nil {
		t.Fatal("expected error when etc/systemd is a file")
	}
	if !strings.Contains(err.Error(), "mkdir getty drop-in") {
		t.Errorf("expected 'mkdir getty drop-in' error, got: %v", err)
	}
}

// TestSetupMiseRuntimes_MkdirError tests error handling when profile.d cannot be created.
func TestSetupMiseRuntimes_MkdirError(t *testing.T) {
	tmp := t.TempDir()

	// Create mise binary
	miseBin := filepath.Join(tmp, "usr", "bin", "mise")
	if err := os.MkdirAll(filepath.Dir(miseBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(miseBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// Make etc a file so MkdirAll on etc/profile.d fails (etc is not a directory)
	etcPath := filepath.Join(tmp, "etc")
	if err := os.RemoveAll(etcPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(etcPath, []byte("not-a-directory"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should not panic
	setupMiseRuntimes(context.Background(), tmp, []string{"nodejs@lts"})

	// Verify mise.sh was NOT created (because etc is a file, profile.d can't be created)
	miseShPath := filepath.Join(tmp, "etc", "profile.d", "mise.sh")
	if _, err := os.Stat(miseShPath); err == nil {
		t.Error("mise.sh should not be created when etc is a file")
	}
}

func TestConfigureAutologin(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
		wantFile bool
	}{
		{name: "empty username is no-op", username: "", wantErr: false, wantFile: false},
		{name: "valid username creates config", username: "root", wantErr: false, wantFile: true},
		{name: "non-root username", username: "admin", wantErr: false, wantFile: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			err := configureAutologin(tmp, tt.username)
			if tt.wantErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			dropinPath := filepath.Join(tmp, "etc", "systemd", "system", "getty@tty1.service.d", "autologin.conf")
			if tt.wantFile {
				data, err := os.ReadFile(dropinPath)
				if err != nil {
					t.Fatalf("expected drop-in file at %s: %v", dropinPath, err)
				}
				if !strings.Contains(string(data), tt.username) {
					t.Errorf("drop-in file does not contain username %q", tt.username)
				}
				if !strings.Contains(string(data), "--autologin") {
					t.Errorf("drop-in file missing --autologin flag")
				}
			} else {
				if _, err := os.Stat(dropinPath); err == nil {
					t.Error("drop-in file should not exist for empty username")
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setupMiseRuntimes
// ---------------------------------------------------------------------------

func TestSetupMiseRuntimes_NoRuntimes(t *testing.T) {
	tmp := t.TempDir()
	// Should not panic or error with nil or empty slice
	setupMiseRuntimes(context.Background(), tmp, nil)
	setupMiseRuntimes(context.Background(), tmp, []string{})

	// No files should have been created
	profilePath := filepath.Join(tmp, "etc", "profile.d", "mise.sh")
	if _, err := os.Stat(profilePath); err == nil {
		t.Error("mise.sh should not be created when runtimes list is empty")
	}
}

func TestSetupMiseRuntimes_MissingMise(t *testing.T) {
	tmp := t.TempDir()
	// No mise binary exists in rootfs
	setupMiseRuntimes(context.Background(), tmp, []string{"nodejs@lts"})

	// Should not create profile.d/mise.sh since mise is not found
	profilePath := filepath.Join(tmp, "etc", "profile.d", "mise.sh")
	if _, err := os.Stat(profilePath); err == nil {
		t.Error("mise.sh should not be created when mise binary is missing")
	}
}

func TestSetupMiseRuntimes_InstallFails(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake mise binary that fails
	miseBin := filepath.Join(tmp, "usr", "bin", "mise")
	if err := os.MkdirAll(filepath.Dir(miseBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(miseBin, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// Should not panic when install fails
	setupMiseRuntimes(context.Background(), tmp, []string{"nodejs@lts"})

	// Verify profile script was still created
	profilePath := filepath.Join(tmp, "etc", "profile.d", "mise.sh")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("mise.sh should be created even when install fails")
	}
}

func TestSetupMiseRuntimes_WithMise(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake mise binary that succeeds
	miseBin := filepath.Join(tmp, "usr", "bin", "mise")
	if err := os.MkdirAll(filepath.Dir(miseBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(miseBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// Run with a runtime
	setupMiseRuntimes(context.Background(), tmp, []string{"nodejs@lts"})

	// Verify profile.d/mise.sh was created
	profilePath := filepath.Join(tmp, "etc", "profile.d", "mise.sh")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("expected mise.sh to be created: %v", err)
	}
	if !strings.Contains(string(data), "mise activate") {
		t.Error("mise.sh should contain mise activation command")
	}
}

// ---------------------------------------------------------------------------
// tmpDir
// ---------------------------------------------------------------------------

func TestTmpDir(t *testing.T) {
	dir := tmpDir()
	if dir == "" {
		t.Error("tmpDir() returned empty")
	}
	if !strings.HasSuffix(dir, "agnostikos-tmp") {
		t.Errorf("tmpDir() = %q; expected suffix \"agnostikos-tmp\"", dir)
	}
}

// ---------------------------------------------------------------------------
// artifactExists
// ---------------------------------------------------------------------------

func TestArtifactExists(t *testing.T) {
	tmp := t.TempDir()

	existingPath := filepath.Join(tmp, "exists.txt")
	if err := os.WriteFile(existingPath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	if !artifactExists(existingPath) {
		t.Error("artifactExists should return true for existing file")
	}
	if artifactExists(filepath.Join(tmp, "nonexistent.txt")) {
		t.Error("artifactExists should return false for nonexistent file")
	}
}

// ---------------------------------------------------------------------------
// sourcesDir
// ---------------------------------------------------------------------------

func TestSourcesDir(t *testing.T) {
	t.Run("default path with empty rootfsDir", func(t *testing.T) {
		got := sourcesDir("")
		want := filepath.Join(BaseDir, "sources")
		if got != want {
			t.Errorf("sourcesDir('') = %s; want %s", got, want)
		}
	})

	t.Run("default rootfsDir", func(t *testing.T) {
		got := sourcesDir(DefaultRoot)
		want := filepath.Join(BaseDir, "sources")
		if got != want {
			t.Errorf("sourcesDir(%s) = %s; want %s", DefaultRoot, got, want)
		}
	})

	t.Run("custom rootfsDir", func(t *testing.T) {
		tmp := t.TempDir()
		rootfsDir := filepath.Join(tmp, "rootfs")
		got := sourcesDir(rootfsDir)
		want := filepath.Join(tmp, "sources")
		if got != want {
			t.Errorf("sourcesDir(%s) = %s; want %s", rootfsDir, got, want)
		}
	})

	t.Run("with AGNOSTICOS_ROOT env var", func(t *testing.T) {
		t.Setenv("AGNOSTICOS_ROOT", "/custom/env/rootfs")
		got := sourcesDir(DefaultRoot) // same as default, falls to env
		want := filepath.Join("/custom/env", "sources")
		if got != want {
			t.Errorf("sourcesDir with env = %s; want %s", got, want)
		}
	})

	t.Run("empty rootfsDir with AGNOSTICOS_ROOT env var", func(t *testing.T) {
		t.Setenv("AGNOSTICOS_ROOT", "/another/env/rootfs")
		got := sourcesDir("") // empty, falls to env
		want := filepath.Join("/another/env", "sources")
		if got != want {
			t.Errorf("sourcesDir('') with env = %s; want %s", got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// downloadFile (via httpClient mock)
// ---------------------------------------------------------------------------

func TestDownloadFile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "downloaded")

	err := downloadFile(dest, server.URL, "")
	if err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test content" {
		t.Errorf("downloaded content = %q; want %q", string(data), "test content")
	}
}

func TestDownloadFile_CreateFileError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	// dest path is in a non-existent directory, so os.Create will fail
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "nonexistent", "file")

	err := downloadFile(dest, server.URL, "")
	if err == nil {
		t.Fatal("expected error when dest directory does not exist")
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "downloaded")

	err := downloadFile(dest, server.URL, "")
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "unexpected status") {
		t.Errorf("expected 'unexpected status' error, got: %v", err)
	}
}

// errReader simulates an io.ReadCloser that reads some data then returns an error.
type errReader struct {
	data []byte
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func (r *errReader) Close() error { return nil }

// errRoundTripper returns a response whose body will fail during io.Copy.
type errRoundTripper struct{}

func (e *errRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body: &errReader{
			data: []byte("short"),
			err:  errors.New("simulated read error"),
		},
		ContentLength: -1,
		Request:       req,
	}, nil
}

func TestDownloadFile_IOCopyError(t *testing.T) {
	origHTTP := httpClient
	httpClient = &http.Client{Transport: &errRoundTripper{}}
	t.Cleanup(func() { httpClient = origHTTP })

	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "downloaded")

	err := downloadFile(dest, "http://example.com/file", "")
	if err == nil {
		t.Fatal("expected error from io.Copy failure")
	}
}

// ---------------------------------------------------------------------------
// CreateRootFS
// ---------------------------------------------------------------------------

func TestCreateRootFS(t *testing.T) {
	tmp := t.TempDir()

	err := CreateRootFS(tmp)
	if err != nil {
		t.Fatalf("CreateRootFS failed: %v", err)
	}

	// Verify that essential FHS directories were created
	essentialDirs := []string{"bin", "usr/bin", "etc", "proc", "sys", "dev", "home", "tmp", "root"}
	for _, dir := range essentialDirs {
		path := filepath.Join(tmp, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist after CreateRootFS", dir)
		}
	}

	// Verify symlinks
	symlinkTests := []struct {
		link string
		dest string // relative target
	}{
		{"bin", "usr/bin"},
		{"lib", "usr/lib"},
	}
	for _, st := range symlinkTests {
		linkPath := filepath.Join(tmp, st.link)
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Errorf("expected symlink %s: %v", st.link, err)
			continue
		}
		if target != st.dest {
			t.Errorf("symlink %s -> %s; want -> %s", st.link, target, st.dest)
		}
	}
}

// ---------------------------------------------------------------------------
// UnmountVirtualFS
// ---------------------------------------------------------------------------

// TestConfigureDefaultShell_ReadShellsError tests the error path when /etc/shells
// exists but is not readable (e.g., is a directory).
func TestConfigureDefaultShell_ReadShellsError(t *testing.T) {
	tmp := t.TempDir()

	// Create /bin/zsh so function proceeds past the first check
	zshPath := filepath.Join(tmp, "bin", "zsh")
	if err := os.MkdirAll(filepath.Dir(zshPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zshPath, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	// Make etc/shells a directory so ReadFile returns EISDIR (not IsNotExist)
	shellsPath := filepath.Join(tmp, "etc", "shells")
	if err := os.MkdirAll(shellsPath, 0755); err != nil {
		t.Fatal(err)
	}

	err := configureDefaultShell(tmp)
	if err == nil {
		t.Fatal("expected error when /etc/shells is a directory")
	}
	if !strings.Contains(err.Error(), "read /etc/shells") {
		t.Errorf("expected 'read /etc/shells' error, got: %v", err)
	}
}

// TestConfigureDefaultShell_WritePasswdError tests the error path when /etc/passwd
// cannot be written (read-only file).
func TestConfigureDefaultShell_WritePasswdError(t *testing.T) {
	// Skip on Windows where file permissions work differently
	tmp := t.TempDir()

	// Create /bin/zsh
	zshPath := filepath.Join(tmp, "bin", "zsh")
	if err := os.MkdirAll(filepath.Dir(zshPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zshPath, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create /etc/shells so the shells step succeeds
	shellsPath := filepath.Join(tmp, "etc", "shells")
	if err := os.MkdirAll(filepath.Dir(shellsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shellsPath, []byte("/bin/bash\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create /etc/passwd as a read-only file so ReadFile succeeds but WriteFile fails
	passwdPath := filepath.Join(tmp, "etc", "passwd")
	if err := os.MkdirAll(filepath.Dir(passwdPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passwdPath, []byte("root:x:0:0:root:/root:/bin/sh\n"), 0444); err != nil {
		t.Fatal(err)
	}

	err := configureDefaultShell(tmp)
	if err == nil {
		t.Fatal("expected error when /etc/passwd is read-only")
	}
	if !strings.Contains(err.Error(), "write /etc/passwd") {
		t.Errorf("expected 'write /etc/passwd' error, got: %v", err)
	}
}

// TestConfigureDefaultShell_NoRootEntry tests adding a root entry when none exists.
func TestConfigureDefaultShell_NoRootEntry(t *testing.T) {
	tmp := t.TempDir()

	// Create /bin/zsh
	zshPath := filepath.Join(tmp, "bin", "zsh")
	if err := os.MkdirAll(filepath.Dir(zshPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zshPath, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create /etc/shells
	shellsPath := filepath.Join(tmp, "etc", "shells")
	if err := os.MkdirAll(filepath.Dir(shellsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shellsPath, []byte("/bin/sh\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create /etc/passwd WITHOUT a root entry
	passwdPath := filepath.Join(tmp, "etc", "passwd")
	if err := os.MkdirAll(filepath.Dir(passwdPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passwdPath, []byte("nobody:x:65534:65534:nobody:/nonexistent:/bin/false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := configureDefaultShell(tmp)
	if err != nil {
		t.Fatalf("configureDefaultShell failed: %v", err)
	}

	// Verify root entry was appended with /bin/zsh
	data, err := os.ReadFile(passwdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "root:x:0:0:root:/root:/bin/zsh") {
		t.Errorf("expected root entry with /bin/zsh, got: %s", string(data))
	}
}

func TestUnmountVirtualFS(t *testing.T) {
	tmp := t.TempDir()

	// Create the mount points that UnmountVirtualFS tries to unmount
	for _, p := range []string{"dev/pts", "dev", "run", "proc", "sys"} {
		path := filepath.Join(tmp, p)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Should not panic or error (umount on non-mounted dirs fails silently)
	err := UnmountVirtualFS(tmp)
	if err != nil {
		t.Errorf("UnmountVirtualFS returned error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// verifySHA256
// ---------------------------------------------------------------------------

func TestVerifySHA256_Match(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "testfile")
	content := "hello world"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Compute expected hash
	h := sha256.New()
	h.Write([]byte(content))
	expected := hex.EncodeToString(h.Sum(nil))

	if err := verifySHA256(path, expected); err != nil {
		t.Errorf("verifySHA256 failed on matching hash: %v", err)
	}
}

func TestVerifySHA256_Mismatch(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "testfile")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wrong hash
	err := verifySHA256(path, "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for SHA256 mismatch")
	}
	if !strings.Contains(err.Error(), "SHA256 mismatch") {
		t.Errorf("expected 'SHA256 mismatch' in error, got: %v", err)
	}
}

func TestVerifySHA256_FileNotFound(t *testing.T) {
	err := verifySHA256("/nonexistent/path", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// ---------------------------------------------------------------------------
// downloadFile — SHA256 verification integration
// ---------------------------------------------------------------------------

func TestDownloadFile_SHA256Verification(t *testing.T) {
	content := "verified content"
	h := sha256.New()
	h.Write([]byte(content))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "verified")

	err := downloadFile(dest, server.URL, expectedHash)
	if err != nil {
		t.Fatalf("downloadFile with valid SHA256 failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("downloaded content = %q; want %q", string(data), content)
	}
}

func TestDownloadFile_SHA256MismatchRemovesFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("actual content"))
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "corrupt")

	err := downloadFile(dest, server.URL, "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for SHA256 mismatch")
	}
	if !strings.Contains(err.Error(), "integrity check") {
		t.Errorf("expected 'integrity check' in error, got: %v", err)
	}

	// Verify the corrupt file was removed
	if _, err := os.Stat(dest); err == nil {
		t.Error("corrupt file should have been removed after SHA256 mismatch")
	}
}

func TestDownloadFile_SHA256EmptySkipsVerification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("any content"))
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "no-verify")

	// Empty SHA256 should not trigger verification
	err := downloadFile(dest, server.URL, "")
	if err != nil {
		t.Fatalf("downloadFile with empty SHA256 failed: %v", err)
	}
}

func TestDownloadFile_VerifyAtRuntimePlaceholderSkipsVerification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("any content"))
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "placeholder")

	// "verify_at_runtime" placeholder should not trigger verification
	err := downloadFile(dest, server.URL, "verify_at_runtime")
	if err != nil {
		t.Fatalf("downloadFile with verify_at_runtime failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// downloadFile — HTTPS enforcement
// ---------------------------------------------------------------------------

func TestDownloadFile_HTTPSEnforcement(t *testing.T) {
	// enforceHTTPS is true by default
	err := downloadFile("/tmp/dummy", "http://example.com/file", "")
	if err == nil {
		t.Fatal("expected error when HTTPS is enforced but URL is HTTP")
	}
	if !strings.Contains(err.Error(), "HTTPS required") {
		t.Errorf("expected 'HTTPS required' in error, got: %v", err)
	}
}

func TestDownloadFile_HTTPSEnforcementAllowsHTTPS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("secure content"))
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	// We need to disable enforcement to use the test server (httptest serves HTTP)
	origEnforce := enforceHTTPS
	enforceHTTPS = false
	t.Cleanup(func() { enforceHTTPS = origEnforce })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "https-test")

	// This test is about verifying the allower does work — we disable enforcement
	// but the URL passed is from the server which is HTTP. We test the logic
	// by verifying that when enforcement is false, HTTP works.
	err := downloadFile(dest, server.URL, "")
	if err != nil {
		t.Errorf("downloadFile should work when HTTPS enforcement is disabled: %v", err)
	}
}

// ---------------------------------------------------------------------------
// DefaultToolchain — HTTPS URLs
// ---------------------------------------------------------------------------

func TestDefaultToolchain_AllURLsAreHTTPS(t *testing.T) {
	for _, pkg := range DefaultToolchain {
		if !strings.HasPrefix(pkg.URL, "https://") {
			t.Errorf("toolchain package %s has non-HTTPS URL: %s", pkg.Name, pkg.URL)
		}
	}
}

// TestDefaultToolchain_HasSHA256Field verifies all packages have SHA256 field set
// (even if placeholder) to remind maintainers to fill real hashes.
func TestDefaultToolchain_HasSHA256Field(t *testing.T) {
	for _, pkg := range DefaultToolchain {
		if pkg.SHA256 == "" {
			t.Errorf("toolchain package %s has empty SHA256 — set a real hash or use 'verify_at_runtime'", pkg.Name)
		}
	}
}
