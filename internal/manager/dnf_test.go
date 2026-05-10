package manager

import (
	"errors"
	"testing"
)

// --- DNFBackend table-driven tests ---

func TestDNFBackend_Install(t *testing.T) {
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
			mock:    &MockExecutor{Err: errors.New("dnf: not found")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DNFBackend{exec: tt.mock, bin: "dnf"}
			err := d.Install(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestDNFBackend_Remove(t *testing.T) {
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
			mock:    &MockExecutor{Err: errors.New("remove failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DNFBackend{exec: tt.mock, bin: "dnf"}
			err := d.Remove(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestDNFBackend_Update(t *testing.T) {
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
			mock:    &MockExecutor{Err: errors.New("update failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DNFBackend{exec: tt.mock, bin: "dnf"}
			err := d.Update(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestDNFBackend_UpdateAll(t *testing.T) {
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
			mock:    &MockExecutor{Err: errors.New("update failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DNFBackend{exec: tt.mock, bin: "dnf"}
			err := d.UpdateAll()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateAll() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestDNFBackend_List(t *testing.T) {
	tests := []struct {
		name     string
		mock     *MockExecutor
		wantErr  bool
		wantLen  int
		wantLine string
	}{
		{
			name: "success",
			mock: &MockExecutor{Output: []byte(`Installed Packages
curl.x86_64                       7.76.1-14.fc35         @updates
git.x86_64                        2.31.1-1.fc35          @updates
`)},
			wantErr:  false,
			wantLen:  2,
			wantLine: "curl.x86_64                       7.76.1-14.fc35         @updates",
		},
		{
			name: "no installed packages",
			mock: &MockExecutor{Output: []byte(`Installed Packages
`)},
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
			d := &DNFBackend{exec: tt.mock, bin: "dnf"}
			results, err := d.List()
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

func TestDNFBackend_Search(t *testing.T) {
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
			mock: &MockExecutor{Output: []byte(`=== Name Exactly Matched ===
curl.x86_64 : A utility for getting files from remote servers

=== Name Matched ===
libcurl.x86_64 : A library for getting files from remote servers
`)},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:  "no results",
			query: "nonexistent-pkg-xyz",
			mock: &MockExecutor{Output: []byte(``)},
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
			d := &DNFBackend{exec: tt.mock, bin: "dnf"}
			results, err := d.Search(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(results) != tt.wantCount {
				t.Errorf("Search() got %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestDNFBackend_Name(t *testing.T) {
	d := &DNFBackend{exec: &MockExecutor{}, bin: "dnf"}
	if got := d.Name(); got != "dnf" {
		t.Errorf("Name() = %q, want %q", got, "dnf")
	}
}
