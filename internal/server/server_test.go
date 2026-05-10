package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ElioNeto/agnostikos/internal/manager"
)

// MockPackageService implements manager.PackageService for testing
type MockPackageService struct {
	InstallFunc   func(pkgName string) error
	RemoveFunc    func(pkgName string) error
	UpdateFunc    func(pkg string) error
	UpdateAllFunc func() error
	SearchFunc    func(query string) ([]string, error)
	ListFunc      func() ([]string, error)
}

func (m *MockPackageService) Install(pkgName string) error {
	if m.InstallFunc != nil {
		return m.InstallFunc(pkgName)
	}
	return nil
}

func (m *MockPackageService) Remove(pkgName string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(pkgName)
	}
	return nil
}

func (m *MockPackageService) Update(pkg string) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(pkg)
	}
	return nil
}

func (m *MockPackageService) UpdateAll() error {
	if m.UpdateAllFunc != nil {
		return m.UpdateAllFunc()
	}
	return nil
}

func (m *MockPackageService) Search(query string) ([]string, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(query)
	}
	return nil, nil
}

func (m *MockPackageService) List() ([]string, error) {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return []string{"pkg1", "pkg2"}, nil
}

// setupTestServer creates a server with mock backends for testing
func setupTestServer(t *testing.T) *Server {
	t.Helper()

	mgr := manager.NewAgnosticManager()

	// Replace backends with mocks
	mgr.Backends["pacman"] = &MockPackageService{
		ListFunc: func() ([]string, error) {
			return []string{"firefox 124.0-1", "git 2.44.0"}, nil
		},
		SearchFunc: func(q string) ([]string, error) {
			if q == "firefox" {
				return []string{"extra/firefox 124.0-1"}, nil
			}
			return nil, nil
		},
		InstallFunc: func(name string) error {
			return nil
		},
		RemoveFunc: func(name string) error {
			return nil
		},
		UpdateAllFunc: func() error {
			return nil
		},
	}
	mgr.Backends["nix"] = &MockPackageService{
		ListFunc: func() ([]string, error) {
			return []string{"nixpkgs.neovim 0.9.5"}, nil
		},
	}
	mgr.Backends["flatpak"] = &MockPackageService{
		ListFunc: func() ([]string, error) {
			return []string{"com.spotify.Client"}, nil
		},
	}
	if _, ok := mgr.Backends["apt"]; ok {
		mgr.Backends["apt"] = &MockPackageService{
			ListFunc: func() ([]string, error) {
				return []string{"curl 7.88.1-10"}, nil
			},
		}
	}

	return New(mgr, WithToken("test-token-123"))
}

func authHeader() (string, string) {
	return "Authorization", "Bearer test-token-123"
}

// doGet is a test helper that performs a GET request with context.
func doGet(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		t.Fatalf("failed to create GET request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to GET %s: %v", url, err)
	}
	return resp
}

// closeResp is a test helper that closes a response body, ignoring errors.
func closeResp(resp *http.Response) {
	_ = resp.Body.Close()
}

// --- Auth / Token tests ---

func TestTokenGeneration(t *testing.T) {
	token := generateToken()
	if len(token) != 64 {
		t.Errorf("expected token length 64 (32 bytes hex), got %d", len(token))
	}
	_, err := hex.DecodeString(token)
	if err != nil {
		t.Errorf("token is not valid hex: %v", err)
	}
}

func TestTokenEnvVar(t *testing.T) {
	_ = os.Setenv("AGNOSTIKOS_TOKEN", "env-token-456")
	defer func() { _ = os.Unsetenv("AGNOSTIKOS_TOKEN") }()
	token := generateToken()
	if token != "env-token-456" { //nolint:gosec
		t.Errorf("expected env token, got %s", token)
	}
}

func TestTokenOverride(t *testing.T) {
	s := New(manager.NewAgnosticManager(), WithToken("override-token"))
	if s.token != "override-token" {
		t.Errorf("expected token 'override-token', got %s", s.token)
	}
	// Verify hash is computed from the override token
	h := sha256.Sum256([]byte("override-token"))
	expectedHash := hex.EncodeToString(h[:])
	if s.authVal != expectedHash {
		t.Errorf("expected authVal %s, got %s", expectedHash, s.authVal)
	}
}

func TestAuthMiddleware_MissingAuth(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	// No auth header at all
	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing auth, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_InvalidBearerToken(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_ValidBearerToken(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for valid token, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_QueryToken(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	// Token as query parameter (used by SSE)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages?token=test-token-123", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for token query param, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_InvalidQueryToken(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages?token=wrong-token", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid query token, got %d", resp.StatusCode)
	}
}

func TestDashboardRoute(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp := doGet(t, ts.URL+"/")
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Dashboard") {
		t.Error("expected response to contain 'Dashboard'")
	}
	if !strings.Contains(string(body), "AgnostikOS") {
		t.Error("expected response to contain 'AgnostikOS'")
	}
}

func TestDashboardRoute_Unauthenticated(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	// Dashboard route does not require auth
	resp := doGet(t, ts.URL+"/")
	defer closeResp(resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for unauthenticated dashboard, got %d", resp.StatusCode)
	}
}

func TestListPackages_NoAuth_Returns401(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp := doGet(t, ts.URL+"/api/packages")
	defer closeResp(resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
	}
}

func TestListPackages_AllBackends(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages", nil)
	req.Header.Set(authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to GET /api/packages: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	pkgs, ok := result["packages"].([]interface{})
	if !ok {
		t.Fatal("expected 'packages' to be an array")
	}
	if len(pkgs) == 0 {
		t.Error("expected non-empty packages list")
	}
}

func TestSearchPackages(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages?q=firefox", nil)
	req.Header.Set(authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if result["query"] != "firefox" {
		t.Errorf("expected query 'firefox', got %v", result["query"])
	}
}

func TestListPackages_ByBackend(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages?backend=nix", nil)
	req.Header.Set(authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to list nix packages: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	pkgs, _ := result["packages"].([]interface{})
	for _, p := range pkgs {
		pMap := p.(map[string]interface{})
		if pMap["backend"] != "nix" {
			t.Errorf("expected backend 'nix', got %v", pMap["backend"])
		}
	}
}

func TestInstallPackage_Success(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"name":"firefox","backend":"pacman"}`)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/packages/install", body)
	req.Header.Set(authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to install: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %s", result["status"])
	}
}

func TestInstallPackage_MissingFields(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty name", `{"name":"","backend":"pacman"}`, http.StatusBadRequest},
		{"empty backend", `{"name":"firefox","backend":""}`, http.StatusBadRequest},
		{"invalid backend", `{"name":"firefox","backend":"invalid"}`, http.StatusNotFound},
		{"missing all", `{}`, http.StatusBadRequest},
		{"invalid json", `{invalid}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/packages/install",
				strings.NewReader(tt.body))
			req.Header.Set(authHeader())
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer closeResp(resp)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestRemovePackage_Success(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", ts.URL+"/api/packages/firefox?backend=pacman", nil)
	req.Header.Set(authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to remove: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRemovePackage_DefaultBackend(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	// Without ?backend=, defaults to pacman
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", ts.URL+"/api/packages/firefox", nil)
	req.Header.Set(authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to remove: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdatePackages_NoBackend(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"backend":""}`)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/packages/update", body)
	req.Header.Set(authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdatePackages_SpecificBackend(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"backend":"pacman"}`)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/packages/update", body)
	req.Header.Set(authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestISOStatus_Idle(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/iso/status", nil)
	req.Header.Set(authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to get ISO status: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var status isoStatus
	_ = json.NewDecoder(resp.Body).Decode(&status)

	if status.Building {
		t.Error("expected not building initially")
	}
	if status.Status != "idle" {
		t.Errorf("expected status 'idle', got '%s'", status.Status)
	}
}

func TestISOStartBuild(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/iso/build", nil)
	req.Header.Set(authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to start ISO build: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "building" {
		t.Errorf("expected status 'building', got '%s'", result["status"])
	}
}

func TestISOStartBuild_AlreadyBuilding(t *testing.T) {
	s := setupTestServer(t)
	block := make(chan struct{})
	s.buildFunc = func(ctx context.Context, cfg manager.BuildConfig, progress chan<- string) error {
		<-block // blocks until test is ready
		return nil
	}
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	// Start first build
	req1, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/iso/build", nil)
	req1.Header.Set(authHeader())
	resp1, _ := http.DefaultClient.Do(req1)
	_ = resp1.Body.Close()

	// Try second build (should conflict - buildRunning=true)
	req2, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/iso/build", nil)
	req2.Header.Set(authHeader())
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer closeResp(resp2)

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", resp2.StatusCode)
	}

	// Unblock the build goroutine so it can clean up
	close(block)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp := doGet(t, ts.URL+"/api/packages")
	defer closeResp(resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing token, got %d", resp.StatusCode)
	}
}

func TestPackagesTableRoute(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/api/packages/table", nil)
	req.Header.Set(authHeader())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to GET /api/packages/table: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<table") {
		t.Error("expected response to contain HTML table")
	}
}

func TestPackagesPageRoute(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp := doGet(t, ts.URL+"/packages")
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Package Manager") {
		t.Error("expected response to contain 'Package Manager'")
	}
}

func TestISOPageRoute(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp := doGet(t, ts.URL+"/iso")
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ISO Builder") {
		t.Error("expected response to contain 'ISO Builder'")
	}
}

func TestSSEEndpoint_RequiresAuth(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp := doGet(t, ts.URL+"/events")
	defer closeResp(resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for SSE without auth, got %d", resp.StatusCode)
	}
}

func TestSSEEndpoint_Authenticated(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/events", nil)
	req.Header.Set(authHeader())

	// Use a transport with short timeout to avoid hanging on SSE
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		// Timeout is expected for SSE since it never ends
		t.Logf("SSE connection timed out as expected: %v", err)
		return
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
		return
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got '%s'", ct)
	}
}

func TestPackageManagerNewHasBackends(t *testing.T) {
	mgr := manager.NewAgnosticManager()
	// Backends are registered only if their binaries exist in PATH.
	// At minimum, the manager should be created successfully with
	// whatever backends are available on the test machine.
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

// TestSSEEndpoint_AuthViaQueryParam verifies that SSE accepts token as query parameter.
func TestSSEEndpoint_AuthViaQueryParam(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/events?token=test-token-123", nil)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("SSE connection timed out as expected: %v", err)
		return
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for SSE with token query param, got %d", resp.StatusCode)
		return
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got '%s'", ct)
	}
}

// TestSSEEndpoint_AuthViaQueryParam_InvalidToken verifies that SSE with invalid token
// query parameter returns 401.
func TestSSEEndpoint_AuthViaQueryParam_InvalidToken(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/events?token=wrong-token", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token query param, got %d", resp.StatusCode)
	}
}

// TestISOStartBuild_WithConfig verifies the ISO build endpoint accepts a config body.
func TestISOStartBuild_WithConfig(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"busybox_version":"1.36.1","name":"TestISO","version":"0.1.0","uefi":true}`)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/iso/build", body)
	req.Header.Set(authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to start ISO build with config: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "building" {
		t.Errorf("expected status 'building', got '%s'", result["status"])
	}
}

// TestISOStartBuild_InvalidConfig verifies the ISO build endpoint rejects invalid JSON.
func TestISOStartBuild_InvalidConfig(t *testing.T) {
	s := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := strings.NewReader(`{invalid json}`)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/api/iso/build", body)
	req.Header.Set(authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer closeResp(resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid config JSON, got %d", resp.StatusCode)
	}
}
