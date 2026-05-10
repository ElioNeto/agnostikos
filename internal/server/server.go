// Package server provides an HTTP interface for the AgnostikOS package manager,
// including a web dashboard, package management API, ISO build triggers, and
// server-sent events for real-time progress updates.
package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ElioNeto/agnostikos/internal/manager"
)

//go:embed templates/*.html
var templateFS embed.FS

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// ServerOption configures the Server.
type ServerOption func(*Server)

// WithToken sets a fixed auth token instead of auto-generating one.
func WithToken(token string) ServerOption {
	return func(s *Server) {
		s.token = token
		h := sha256.Sum256([]byte(token))
		s.authVal = hex.EncodeToString(h[:])
	}
}

// WithTLS enables HTTPS using the given certificate and key files.
func WithTLS(certFile, keyFile string) ServerOption {
	return func(s *Server) {
		s.tlsCert = certFile
		s.tlsKey = keyFile
	}
}

// Server wraps the AgnostikOS manager with an HTTP interface
type Server struct {
	mgr     *manager.AgnosticManager
	mux     *http.ServeMux
	tmpl    *template.Template
	token   string // raw token for display
	authVal string // hex-encoded SHA-256 of auth token

	// TLS
	tlsCert string
	tlsKey  string

	// SSE
	progress    string
	progressMu  sync.RWMutex
	buildRunning bool    // separate flag set before goroutine, cleared after
	buildFunc    func(ctx context.Context, cfg manager.BuildConfig, progress chan<- string) error
	sseChannels map[chan SSEEvent]struct{}
	sseMu       sync.Mutex
}

// New creates a new Server
func New(mgr *manager.AgnosticManager, opts ...ServerOption) *Server {
	s := &Server{
		mgr:         mgr,
		mux:         http.NewServeMux(),
		sseChannels: make(map[chan SSEEvent]struct{}),
	}
	s.buildFunc = func(ctx context.Context, cfg manager.BuildConfig, progress chan<- string) error {
		return s.mgr.Build(ctx, cfg, progress)
	}

	// Parse templates
	s.tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

	// Apply options first (WithToken may set token)
	for _, opt := range opts {
		opt(s)
	}

	// Auth token — if not set via option, fallback to env var or auto-generate
	if s.token == "" {
		s.token = generateToken()
		h := sha256.Sum256([]byte(s.token))
		s.authVal = hex.EncodeToString(h[:])
	}
	fmt.Printf("AGNOSTIC TOKEN: %s\n", s.token)

	// Routes — Go 1.22+ ServeMux with method+pattern routing
	s.mux.HandleFunc("GET /", s.handleDashboard)
	s.mux.HandleFunc("GET /packages", s.handlePackagesPage)
	s.mux.HandleFunc("GET /iso", s.handleISOPage)
	s.mux.HandleFunc("GET /api/packages", s.withAuth(s.handleListPackages))
	s.mux.HandleFunc("GET /api/packages/table", s.withAuth(s.handlePackagesTable))
	s.mux.HandleFunc("POST /api/packages/install", s.withAuth(s.handleInstallPackage))
	s.mux.HandleFunc("DELETE /api/packages/{name}", s.withAuth(s.handleRemovePackage))
	s.mux.HandleFunc("POST /api/packages/update", s.withAuth(s.handleUpdatePackages))
	s.mux.HandleFunc("GET /api/iso/status", s.withAuth(s.handleISOStatus))
	s.mux.HandleFunc("POST /api/iso/build", s.withAuth(s.handleISOStartBuild))
	s.mux.HandleFunc("GET /events", s.withAuth(s.handleSSE))

	return s
}

// Listen starts the HTTP or HTTPS server on the given address.
// TLS is used when both --tls-cert and --tls-key are configured via WithTLS.
func (s *Server) Listen(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if s.tlsCert != "" && s.tlsKey != "" {
		log.Printf("Server starting on https://%s", addr)
		return srv.ListenAndServeTLS(s.tlsCert, s.tlsKey)
	}

	log.Printf("Server starting on http://%s", addr)
	return srv.ListenAndServe()
}

// Handler returns the HTTP handler (for testing)
func (s *Server) Handler() http.Handler {
	return s.mux
}

// generateToken returns a token from env var or auto-generates 32 random bytes.
func generateToken() string {
	if token := os.Getenv("AGNOSTIKOS_TOKEN"); token != "" {
		return token
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("failed to generate auth token: %v", err)
	}
	return hex.EncodeToString(buf)
}

// withAuth wraps a handler to require authentication via Authorization: Bearer header
// or token query parameter (the latter is required for SSE via EventSource).
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := ""

		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}

		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token == "" {
			http.Error(w, "Missing Authorization: Bearer header or token query parameter", http.StatusUnauthorized)
			return
		}
		h := sha256.Sum256([]byte(token))
		if hex.EncodeToString(h[:]) != s.authVal {
			http.Error(w, "Invalid auth token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// --- Dashboard ---

type dashboardData struct {
	Backends []string
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := dashboardData{
		Backends: s.mgr.ListBackends(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("template error: %v", err)
	}
}

// --- Packages Page ---

func (s *Server) handlePackagesPage(w http.ResponseWriter, r *http.Request) {
	data := dashboardData{
		Backends: s.mgr.ListBackends(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("template error: %v", err)
	}
}

// --- ISO Page ---

func (s *Server) handleISOPage(w http.ResponseWriter, r *http.Request) {
	data := dashboardData{
		Backends: s.mgr.ListBackends(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("template error: %v", err)
	}
}

// --- List / Search Packages ---

type packageResult struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
}

func (s *Server) handleListPackages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	backendName := r.URL.Query().Get("backend")

	var packages []packageResult

	if backendName == "" {
		for name := range s.mgr.Backends {
			pkgs, err := s.listFromBackend(r.Context(), name, q)
			if err != nil {
				log.Printf("error listing backend %s: %v", name, err)
				continue
			}
			packages = append(packages, pkgs...)
		}
	} else {
		pkgs, err := s.listFromBackend(r.Context(), backendName, q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		packages = pkgs
	}

	resp := map[string]interface{}{
		"packages": packages,
	}
	if q != "" {
		resp["query"] = q
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) listFromBackend(ctx context.Context, name, query string) ([]packageResult, error) {
	svc, ok := s.mgr.Backends[name]
	if !ok {
		return nil, fmt.Errorf("backend '%s' not found", name)
	}

	var pkgs []string
	var err error
	if query != "" {
		pkgs, err = svc.Search(query)
	} else {
		pkgs, err = svc.List()
	}
	if err != nil {
		return nil, fmt.Errorf("backend %s: %w", name, err)
	}

	results := make([]packageResult, 0, len(pkgs))
	for _, p := range pkgs {
		results = append(results, packageResult{Name: p, Backend: name})
	}
	return results, nil
}

// --- Packages Table (HTML fragment for HTMX) ---

func (s *Server) handlePackagesTable(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	backendName := r.URL.Query().Get("backend")

	var packages []packageResult

	if backendName == "" {
		for name := range s.mgr.Backends {
			pkgs, err := s.listFromBackend(r.Context(), name, q)
			if err != nil {
				log.Printf("error listing backend %s: %v", name, err)
				continue
			}
			packages = append(packages, pkgs...)
		}
	} else {
		pkgs, err := s.listFromBackend(r.Context(), backendName, q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		packages = pkgs
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(packages) == 0 {
		_, _ = w.Write([]byte(`<p style="color:#64748b;">No packages found.</p>`))
		return
	}

	_, _ = w.Write([]byte(`<table><thead><tr><th>Package</th><th>Backend</th><th>Actions</th></tr></thead><tbody>`))
	for _, p := range packages {
		escapedName := template.HTMLEscapeString(p.Name)
		escapedBackend := template.HTMLEscapeString(p.Backend)
		_, _ = fmt.Fprintf(w, `<tr>
			<td>%s</td>
			<td><span class="badge badge-%s">%s</span></td>
			<td class="flex" style="gap:0.5rem;">
				<button onclick="installPackage('%s','%s')">Install</button>
				<button class="danger" onclick="removePackage('%s','%s')">Remove</button>
			</td>
		</tr>`, escapedName, escapedBackend, escapedBackend, escapedName, escapedBackend, escapedName, escapedBackend)
	}
	_, _ = w.Write([]byte(`</tbody></table>`))
}

// --- Install Package ---

type installRequest struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
}

func (s *Server) handleInstallPackage(w http.ResponseWriter, r *http.Request) {
	var req installRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Backend == "" {
		http.Error(w, "name and backend are required", http.StatusBadRequest)
		return
	}

	svc, ok := s.mgr.Backends[req.Backend]
	if !ok {
		http.Error(w, fmt.Sprintf("backend '%s' not found", req.Backend), http.StatusNotFound)
		return
	}

	if err := svc.Install(req.Name); err != nil {
		s.broadcast(SSEEvent{Event: "install:error", Data: map[string]string{
			"package": req.Name,
			"backend": req.Backend,
			"error":   err.Error(),
		}})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.broadcast(SSEEvent{Event: "install:done", Data: map[string]string{
		"package": req.Name,
		"backend": req.Backend,
	}})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "package": req.Name})
}

// --- Remove Package ---

func (s *Server) handleRemovePackage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	backendName := r.URL.Query().Get("backend")
	if backendName == "" {
		backendName = "pacman"
	}

	svc, ok := s.mgr.Backends[backendName]
	if !ok {
		http.Error(w, fmt.Sprintf("backend '%s' not found", backendName), http.StatusNotFound)
		return
	}

	if err := svc.Remove(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "package": name})
}

// --- Update Packages ---

type updateRequest struct {
	Backend string `json:"backend"`
}

// updateBackend calls UpdateAll on the named backend.
func (s *Server) updateBackend(ctx context.Context, name string) error {
	svc, ok := s.mgr.Backends[name]
	if !ok {
		return fmt.Errorf("backend '%s' not found", name)
	}

	s.broadcast(SSEEvent{Event: "update:start", Data: map[string]string{
		"backend": name,
	}})

	if err := svc.UpdateAll(); err != nil {
		s.broadcast(SSEEvent{Event: "update:error", Data: map[string]string{
			"backend": name,
			"error":   err.Error(),
		}})
		return err
	}

	s.broadcast(SSEEvent{Event: "update:done", Data: map[string]string{
		"backend": name,
	}})

	return nil
}

func (s *Server) handleUpdatePackages(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Backend == "" {
		for name := range s.mgr.Backends {
			if err := s.updateBackend(r.Context(), name); err != nil {
				s.broadcast(SSEEvent{Event: "update:error", Data: map[string]string{
					"backend": name,
					"error":   err.Error(),
				}})
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		if err := s.updateBackend(r.Context(), req.Backend); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- ISO Status ---

type isoStatus struct {
	Building bool   `json:"building"`
	Status   string `json:"status"`
	Progress string `json:"progress,omitempty"`
}

func (s *Server) handleISOStatus(w http.ResponseWriter, r *http.Request) {
	s.progressMu.RLock()
	status := isoStatus{
		Building: s.progress != "",
		Status:   "idle",
		Progress: s.progress,
	}
	if s.progress != "" {
		status.Status = "building"
	}
	s.progressMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// --- ISO Build ---

// buildConfigWithDefaults applies default values to a BuildConfig.
func buildConfigWithDefaults(cfg manager.BuildConfig) manager.BuildConfig {
	if cfg.BusyboxVersion == "" {
		cfg.BusyboxVersion = "1.36.1"
	}
	if cfg.Name == "" {
		cfg.Name = "AgnostikOS"
	}
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}
	return cfg
}

func (s *Server) handleISOStartBuild(w http.ResponseWriter, r *http.Request) {
	s.progressMu.Lock()
	if s.buildRunning {
		s.progressMu.Unlock()
		http.Error(w, "ISO build already in progress", http.StatusConflict)
		return
	}
	s.buildRunning = true
	s.progress = "starting"
	s.progressMu.Unlock()

	// Parse optional build config from request body
	var cfg manager.BuildConfig
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			s.progressMu.Lock()
			s.progress = ""
			s.progressMu.Unlock()
			http.Error(w, "Invalid build configuration: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	cfg = buildConfigWithDefaults(cfg)

	go func() {
		defer func() {
			s.progressMu.Lock()
			s.buildRunning = false
			s.progress = ""
			s.progressMu.Unlock()
		}()

		s.progressMu.Lock()
		s.progress = "building ISO..."
		s.progressMu.Unlock()

		s.broadcast(SSEEvent{Event: "iso:progress", Data: "Starting bootstrap..."})
		s.broadcast(SSEEvent{Event: "iso:progress", Data: "Building ISO image, please wait..."})

		err := s.buildFunc(context.Background(), cfg, nil)

		if err != nil {
			s.broadcast(SSEEvent{Event: "iso:error", Data: err.Error()})
			return
		}

		s.broadcast(SSEEvent{Event: "iso:done", Data: "ISO build complete"})
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "building"})
}

// --- SSE ---

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan SSEEvent, 64)
	s.sseMu.Lock()
	s.sseChannels[ch] = struct{}{}
	s.sseMu.Unlock()

	defer func() {
		s.sseMu.Lock()
		delete(s.sseChannels, ch)
		s.sseMu.Unlock()
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(evt.Data)
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Event, data)
			flusher.Flush()
		}
	}
}

func (s *Server) broadcast(evt SSEEvent) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for ch := range s.sseChannels {
		select {
		case ch <- evt:
		default:
		}
	}
}
