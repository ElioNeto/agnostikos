package bootstrap

import (
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
		{name: "trailing space no match", content: "/bin/zsh \n", shell: "/bin/zsh", want: false},
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
// configureAutologin
// ---------------------------------------------------------------------------

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
	setupMiseRuntimes(tmp, nil)
	setupMiseRuntimes(tmp, []string{})

	// No files should have been created
	profilePath := filepath.Join(tmp, "etc", "profile.d", "mise.sh")
	if _, err := os.Stat(profilePath); err == nil {
		t.Error("mise.sh should not be created when runtimes list is empty")
	}
}

func TestSetupMiseRuntimes_MissingMise(t *testing.T) {
	tmp := t.TempDir()
	// No mise binary exists in rootfs
	setupMiseRuntimes(tmp, []string{"nodejs@lts"})

	// Should not create profile.d/mise.sh since mise is not found
	profilePath := filepath.Join(tmp, "etc", "profile.d", "mise.sh")
	if _, err := os.Stat(profilePath); err == nil {
		t.Error("mise.sh should not be created when mise binary is missing")
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
	setupMiseRuntimes(tmp, []string{"nodejs@lts"})

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

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "downloaded")

	err := downloadFile(dest, server.URL)
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

func TestDownloadFile_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	origHTTP := httpClient
	httpClient = server.Client()
	t.Cleanup(func() { httpClient = origHTTP })

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "downloaded")

	err := downloadFile(dest, server.URL)
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "unexpected status") {
		t.Errorf("expected 'unexpected status' error, got: %v", err)
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
