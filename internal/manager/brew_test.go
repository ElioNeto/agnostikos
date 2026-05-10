package manager

import (
	"errors"
	"testing"
)

// --- BrewBackend table-driven tests ---

func TestBrewBackend_Install(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		mock    *MockExecutor
		wantErr bool
	}{
		{
			name:    "empty package name",
			pkg:     "",
			mock:    &MockExecutor{},
			wantErr: true,
		},
		{
			name:    "success",
			pkg:     "curl",
			mock:    &MockExecutor{Output: []byte("installed")},
			wantErr: false,
		},
		{
			name:    "exec error",
			pkg:     "curl",
			mock:    &MockExecutor{Err: errors.New("brew: not found")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BrewBackend{exec: tt.mock}
			err := b.Install(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrewBackend_Remove(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		mock    *MockExecutor
		wantErr bool
	}{
		{
			name:    "empty package name",
			pkg:     "",
			mock:    &MockExecutor{},
			wantErr: true,
		},
		{
			name:    "success",
			pkg:     "curl",
			mock:    &MockExecutor{},
			wantErr: false,
		},
		{
			name:    "exec error",
			pkg:     "curl",
			mock:    &MockExecutor{Err: errors.New("uninstall failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BrewBackend{exec: tt.mock}
			err := b.Remove(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrewBackend_Update(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		mock    *MockExecutor
		wantErr bool
	}{
		{
			name:    "empty package name",
			pkg:     "",
			mock:    &MockExecutor{},
			wantErr: true,
		},
		{
			name:    "success",
			pkg:     "curl",
			mock:    &MockExecutor{},
			wantErr: false,
		},
		{
			name:    "exec error",
			pkg:     "curl",
			mock:    &MockExecutor{Err: errors.New("upgrade failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BrewBackend{exec: tt.mock}
			err := b.Update(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrewBackend_UpdateAll(t *testing.T) {
	tests := []struct {
		name    string
		mock    *MockExecutor
		wantErr bool
	}{
		{
			name:    "success",
			mock:    &MockExecutor{},
			wantErr: false,
		},
		{
			name:    "exec error",
			mock:    &MockExecutor{Err: errors.New("upgrade failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BrewBackend{exec: tt.mock}
			err := b.UpdateAll()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateAll() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrewBackend_List(t *testing.T) {
	tests := []struct {
		name     string
		mock     *MockExecutor
		wantErr  bool
		wantLen  int
		wantLine string
	}{
		{
			name: "success",
			mock: &MockExecutor{Output: []byte(`curl
git
wget
`)},
			wantErr:  false,
			wantLen:  3,
			wantLine: "curl",
		},
		{
			name: "no installed packages",
			mock: &MockExecutor{Output: []byte(``)},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "exec error",
			mock:    &MockExecutor{Err: errors.New("list failed")},
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BrewBackend{exec: tt.mock}
			results, err := b.List()
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(results) != tt.wantLen {
				t.Errorf("List() got %d results, want %d", len(results), tt.wantLen)
			}
			if tt.wantLine != "" && len(results) > 0 && results[0] != tt.wantLine {
				t.Errorf("List() first result = %q, want %q", results[0], tt.wantLine)
			}
		})
	}
}

func TestBrewBackend_Search(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		mock      *MockExecutor
		wantErr   bool
		wantCount int
	}{
		{
			name:      "empty query",
			query:     "",
			mock:      &MockExecutor{},
			wantErr:   true,
			wantCount: 0,
		},
		{
			name:  "success",
			query: "curl",
			mock: &MockExecutor{Output: []byte(`==> Formulae
curl
curlftpfs

==> Casks
`)},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:  "no results",
			query: "nonexistent-pkg-xyz",
			mock: &MockExecutor{Output: []byte(`==> Formulae

==> Casks
`)},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:    "exec error",
			query:   "curl",
			mock:    &MockExecutor{Err: errors.New("search failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BrewBackend{exec: tt.mock}
			results, err := b.Search(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(results) != tt.wantCount {
				t.Errorf("Search() got %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestBrewBackend_Name(t *testing.T) {
	b := &BrewBackend{exec: &MockExecutor{}}
	if got := b.Name(); got != "brew" {
		t.Errorf("Name() = %q, want %q", got, "brew")
	}
}
