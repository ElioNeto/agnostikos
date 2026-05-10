package manager

import (
	"errors"
	"testing"
)

// --- AptBackend table-driven tests ---

func TestAptBackend_Install(t *testing.T) {
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
			mock:    &MockExecutor{Err: errors.New("apt-get: not found")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AptBackend{exec: tt.mock}
			err := a.Install(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestAptBackend_Remove(t *testing.T) {
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
			a := &AptBackend{exec: tt.mock}
			err := a.Remove(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestAptBackend_Update(t *testing.T) {
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
			a := &AptBackend{exec: tt.mock}
			err := a.Update(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestAptBackend_UpdateAll(t *testing.T) {
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
			a := &AptBackend{exec: tt.mock}
			err := a.UpdateAll()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateAll() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestAptBackend_List(t *testing.T) {
	tests := []struct {
		name     string
		mock     *MockExecutor
		wantErr  bool
		wantLen  int
		wantLine string
	}{
		{
			name: "success",
			mock: &MockExecutor{Output: []byte(`Desired=Unknown/Install/Remove/Purge/Hold
| Status=Not/Inst/Conf-files/Unpacked/halF-conf/Half-inst/trig-aWait/Trig-pend
|/ Err?=(none)/Reinst-required (Status,Err: uppercase=bad)
||/ Name           Version      Architecture Description
+++-==============-============-============-=================================
ii  curl           7.88.1-10    amd64        command line tool for transferring data
ii  git            1:2.39.5-1   amd64        fast, scalable, distributed revision control
ii  openssh-client 1:9.2p1-2    amd64        secure shell client
`)},
			wantErr:  false,
			wantLen:  3,
			wantLine: "ii  curl           7.88.1-10    amd64        command line tool for transferring data",
		},
		{
			name: "no installed packages",
			mock: &MockExecutor{Output: []byte(`||/ Name Version Architecture Description
+++-==============-============-============-=================================
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
			a := &AptBackend{exec: tt.mock}
			results, err := a.List()
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

func TestAptBackend_Search(t *testing.T) {
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
			mock: &MockExecutor{Output: []byte(`curl - command line tool for transferring data
curlftpfs - filesystem for accessing FTP hosts
libcurl4 - easy-to-use client-side URL transfer library
`)},
			wantErr:   false,
			wantCount: 3,
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
			a := &AptBackend{exec: tt.mock}
			results, err := a.Search(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(results) != tt.wantCount {
				t.Errorf("Search() got %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestAptBackend_Search_ReturnsPackageNames(t *testing.T) {
	output := `curl - command line tool for transferring data
git - fast, scalable, distributed revision control
`
	a := &AptBackend{exec: &MockExecutor{Output: []byte(output)}}
	results, err := a.Search("curl")
	if err != nil {
		t.Fatalf("Search() unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0] != "curl" {
		t.Errorf("expected first result 'curl', got %q", results[0])
	}
	if results[1] != "git" {
		t.Errorf("expected second result 'git', got %q", results[1])
	}
}

func TestAptBackend_Name(t *testing.T) {
	a := &AptBackend{exec: &MockExecutor{}}
	if got := a.Name(); got != "apt" {
		t.Errorf("Name() = %q, want %q", got, "apt")
	}
}
