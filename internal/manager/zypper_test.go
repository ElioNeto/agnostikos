package manager

import (
	"errors"
	"testing"
)

// --- ZypperBackend table-driven tests ---

func TestZypperBackend_Install(t *testing.T) {
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
			mock:    &MockExecutor{Err: errors.New("zypper: not found")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &ZypperBackend{exec: tt.mock}
			err := z.Install(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestZypperBackend_Remove(t *testing.T) {
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
			z := &ZypperBackend{exec: tt.mock}
			err := z.Remove(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestZypperBackend_Update(t *testing.T) {
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
			z := &ZypperBackend{exec: tt.mock}
			err := z.Update(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestZypperBackend_UpdateAll(t *testing.T) {
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
			z := &ZypperBackend{exec: tt.mock}
			err := z.UpdateAll()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateAll() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestZypperBackend_List(t *testing.T) {
	tests := []struct {
		name     string
		mock     *MockExecutor
		wantErr  bool
		wantLen  int
		wantLine string
	}{
		{
			name: "success",
			mock: &MockExecutor{Output: []byte(`S | Name        | Version    | Arch
--+-------------+------------+-------
i | curl        | 7.76.1     | x86_64
i | git         | 2.31.1     | x86_64
`)},
			wantErr:  false,
			wantLen:  2,
			wantLine: "i | curl        | 7.76.1     | x86_64",
		},
		{
			name: "no installed packages",
			mock: &MockExecutor{Output: []byte(`S | Name | Version | Arch
--+------+---------+-----
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
			z := &ZypperBackend{exec: tt.mock}
			results, err := z.List()
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

func TestZypperBackend_Search(t *testing.T) {
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
			mock: &MockExecutor{Output: []byte(`S | Name        | Summary    | Type
--+-------------+------------+-------
  | curl        | Transfer data | package
  | libcurl     | Library    | package
`)},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:  "no results",
			query: "nonexistent-pkg-xyz",
			mock: &MockExecutor{Output: []byte(`S | Name | Summary | Type
--+------+---------+-----
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
			z := &ZypperBackend{exec: tt.mock}
			results, err := z.Search(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(results) != tt.wantCount {
				t.Errorf("Search() got %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestZypperBackend_Name(t *testing.T) {
	z := &ZypperBackend{exec: &MockExecutor{}}
	if got := z.Name(); got != "zypper" {
		t.Errorf("Name() = %q, want %q", got, "zypper")
	}
}
