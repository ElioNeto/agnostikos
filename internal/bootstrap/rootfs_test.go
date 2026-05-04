package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLFSTarget_Arg(t *testing.T) {
	if got := resolveLFSTarget("/custom"); got != "/custom" {
		t.Errorf("expected /custom, got %s", got)
	}
}

func TestResolveLFSTarget_EnvVar(t *testing.T) {
	t.Setenv("LFS", "/from-env")
	if got := resolveLFSTarget(""); got != "/from-env" {
		t.Errorf("expected /from-env, got %s", got)
	}
}

func TestResolveLFSTarget_Default(t *testing.T) {
	os.Unsetenv("LFS")
	if got := resolveLFSTarget(""); got != DefaultLFSRoot {
		t.Errorf("expected %s, got %s", DefaultLFSRoot, got)
	}
}

func TestFHSDirectories_Created(t *testing.T) {
	tmp, err := os.MkdirTemp("", "lfs-fhs-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

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
	defer os.RemoveAll(tmp)

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
	if err := DownloadToolchain(tmp); err != nil {
		t.Errorf("expected no error when files exist, got: %v", err)
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
