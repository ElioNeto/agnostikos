package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ElioNeto/agnostikos/internal/manager"
)

// mockBackend implements manager.PackageService for testing the TUI.
type mockBackend struct {
	installErr error
	removeErr  error
	searchRes  []string
	searchErr  error
}

func (m *mockBackend) Install(pkgName string) error        { return m.installErr }
func (m *mockBackend) Remove(pkgName string) error         { return m.removeErr }
func (m *mockBackend) Update(pkg string) error             { return nil }
func (m *mockBackend) UpdateAll() error                    { return nil }
func (m *mockBackend) Search(q string) ([]string, error)   { return m.searchRes, m.searchErr }
func (m *mockBackend) List() ([]string, error)             { return []string{"pkg1", "pkg2"}, nil }

// newTestModel creates a Model with a mock backend for testing.
func newTestModel() Model {
	mgr := &manager.AgnosticManager{
		Backends: map[string]manager.PackageService{
			"pacman": &mockBackend{
				searchRes: []string{"firefox", "firefox-esr", "firefox-developer"},
			},
			"nix": &mockBackend{
				searchRes: []string{"nixpkgs.firefox", "nixpkgs.firefox-esr"},
			},
			"flatpak": &mockBackend{
				searchRes: []string{"org.mozilla.firefox", "org.mozilla.firefox-esr"},
			},
		},
	}
	return InitialModel(mgr)
}

// sendKey sends a key press message to the model and returns the updated model.
// Handles both regular rune keys and special key names ("up", "down").
func sendKey(m Model, key string) Model {
	var msg tea.KeyMsg
	switch key {
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// sendEnter sends an Enter key press.
func sendEnter(m Model) Model {
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// sendEsc sends an Escape key press.
func sendEsc(m Model) Model {
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// sendRune sends a typed character to the search input.
func sendRune(m Model, r rune) Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
	updated, cmd := m.Update(msg)
	// Process cmd if it's a textinput blink
	if cmd != nil {
		cmd()
	}
	return updated.(Model)
}

// sendSearchResults sends a searchResultsMsg to the model.
func sendSearchResults(m Model, results []string, err error) Model {
	msg := searchResultsMsg{results: results, err: err}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// sendActionCompleted sends an actionCompletedMsg to the model.
func sendActionCompleted(m Model, action, pkg string, err error) Model {
	msg := actionCompletedMsg{action: action, pkg: pkg, err: err}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// --- Tests ---

func TestInitialModel(t *testing.T) {
	m := newTestModel()

	if m.viewState != BackendListView {
		t.Errorf("expected BackendListView, got %v", m.viewState)
	}
	if len(m.backends) != 3 {
		t.Errorf("expected 3 backends, got %d", len(m.backends))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if m.loading {
		t.Error("expected loading to be false initially")
	}
}

func TestBackendListNavigation_Down(t *testing.T) {
	m := newTestModel()

	m = sendKey(m, "j")
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after 'j', got %d", m.cursor)
	}

	m = sendKey(m, "down")
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after 'down', got %d", m.cursor)
	}

	// Should not go past the last item
	m = sendKey(m, "j")
	if m.cursor != 2 {
		t.Errorf("expected cursor to stay at 2 (last item), got %d", m.cursor)
	}
}

func TestBackendListNavigation_Up(t *testing.T) {
	m := newTestModel()
	m.cursor = 2

	m = sendKey(m, "k")
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after 'k', got %d", m.cursor)
	}

	m = sendKey(m, "up")
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 after 'up', got %d", m.cursor)
	}

	// Should not go past the first item
	m = sendKey(m, "k")
	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0 (first item), got %d", m.cursor)
	}
}

func TestBackendListSelect_TransitionsToSearch(t *testing.T) {
	m := newTestModel()

	m = sendEnter(m)
	if m.viewState != SearchView {
		t.Errorf("expected SearchView after enter, got %v", m.viewState)
	}
	if m.searchInput.Value() != "" {
		t.Errorf("expected empty search input, got %q", m.searchInput.Value())
	}
	if m.searchResults != nil {
		t.Errorf("expected nil search results, got %v", m.searchResults)
	}
}

func TestSearchView_TypeAndNavigateResults(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m) // Go to search view

	// Type a query
	m = sendRune(m, 'f')
	if m.searchInput.Value() != "f" {
		t.Errorf("expected search input 'f', got %q", m.searchInput.Value())
	}

	// Receive search results
	m = sendSearchResults(m, []string{"firefox", "firefox-esr"}, nil)

	if len(m.searchResults) != 2 {
		t.Errorf("expected 2 search results, got %d", len(m.searchResults))
	}
	if m.searchErr != nil {
		t.Errorf("expected no search error, got %v", m.searchErr)
	}
	if m.loading {
		t.Error("expected loading to be false after receiving results")
	}
}

func TestSearchResults_NavigateDown(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)
	m = sendSearchResults(m, []string{"firefox", "firefox-esr", "firefox-dev"}, nil)

	// Navigate down
	m = sendKey(m, "j")
	if m.searchCursor != 1 {
		t.Errorf("expected searchCursor 1, got %d", m.searchCursor)
	}

	m = sendKey(m, "down")
	if m.searchCursor != 2 {
		t.Errorf("expected searchCursor 2, got %d", m.searchCursor)
	}

	// Should not go past last
	m = sendKey(m, "down")
	if m.searchCursor != 2 {
		t.Errorf("expected searchCursor to stay at 2, got %d", m.searchCursor)
	}
}

func TestSearchResults_NavigateUp(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)
	m = sendSearchResults(m, []string{"firefox", "firefox-esr", "firefox-dev"}, nil)
	m.searchCursor = 2

	m = sendKey(m, "k")
	if m.searchCursor != 1 {
		t.Errorf("expected searchCursor 1, got %d", m.searchCursor)
	}

	m = sendKey(m, "up")
	if m.searchCursor != 0 {
		t.Errorf("expected searchCursor 0, got %d", m.searchCursor)
	}

	// Should not go past first
	m = sendKey(m, "up")
	if m.searchCursor != 0 {
		t.Errorf("expected searchCursor to stay at 0, got %d", m.searchCursor)
	}
}

func TestSearchSelect_TransitionsToPackageDetail(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)
	m = sendSearchResults(m, []string{"firefox", "firefox-esr"}, nil)

	// Select first result
	m = sendEnter(m)

	if m.viewState != PackageDetailView {
		t.Errorf("expected PackageDetailView, got %v", m.viewState)
	}
	if m.selectedPkg != "firefox" {
		t.Errorf("expected selectedPkg 'firefox', got %q", m.selectedPkg)
	}
}

func TestSearchSelect_SecondItem(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)
	m = sendSearchResults(m, []string{"firefox", "firefox-esr"}, nil)

	// Navigate to second item and select
	m = sendKey(m, "j")
	m = sendEnter(m)

	if m.selectedPkg != "firefox-esr" {
		t.Errorf("expected selectedPkg 'firefox-esr', got %q", m.selectedPkg)
	}
}

func TestPackageDetail_Install(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView

	m = sendKey(m, "i")
	if !m.loading {
		t.Error("expected loading to be true after install")
	}

	// Receive completion
	m = sendActionCompleted(m, "install", "firefox", nil)
	if m.loading {
		t.Error("expected loading to be false after completion")
	}
	if m.actionErr != nil {
		t.Errorf("expected no action error, got %v", m.actionErr)
	}
}

func TestPackageDetail_InstallError(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView

	m = sendKey(m, "i")
	m = sendActionCompleted(m, "install", "firefox", errors.New("permission denied"))

	if m.loading {
		t.Error("expected loading to be false after error")
	}
	if m.actionErr == nil {
		t.Error("expected action error, got nil")
	}
}

func TestPackageDetail_Remove(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView

	m = sendKey(m, "r")
	if !m.loading {
		t.Error("expected loading to be true after remove")
	}

	m = sendActionCompleted(m, "remove", "firefox", nil)
	if m.loading {
		t.Error("expected loading to be false after completion")
	}
}

func TestPackageDetail_RemoveError(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView

	m = sendKey(m, "r")
	m = sendActionCompleted(m, "remove", "firefox", errors.New("not found"))

	if m.actionErr == nil {
		t.Error("expected action error, got nil")
	}
}

func TestPackageDetail_EscapeToSearch(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView

	m = sendEsc(m)
	if m.viewState != SearchView {
		t.Errorf("expected SearchView after escape, got %v", m.viewState)
	}
}

func TestSearchView_EscapeToBackendList(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m) // Go to search
	m = sendEsc(m)   // Back to backend list

	if m.viewState != BackendListView {
		t.Errorf("expected BackendListView after escape, got %v", m.viewState)
	}
}

func TestFullCycle_BackendListToDetailAndBack(t *testing.T) {
	m := newTestModel()

	// Backend list → search
	m = sendEnter(m)

	// Receive some results
	m = sendSearchResults(m, []string{"firefox", "firefox-esr"}, nil)

	// Select first result → detail
	m = sendEnter(m)
	if m.viewState != PackageDetailView {
		t.Fatalf("expected PackageDetailView, got %v", m.viewState)
	}

	// Detail → search (esc)
	m = sendEsc(m)
	if m.viewState != SearchView {
		t.Errorf("expected SearchView, got %v", m.viewState)
	}

	// Search → backend list (esc)
	m = sendEsc(m)
	if m.viewState != BackendListView {
		t.Errorf("expected BackendListView, got %v", m.viewState)
	}
}

func TestEmptySearchResults(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)

	m = sendSearchResults(m, nil, nil)
	if m.searchResults != nil {
		t.Errorf("expected nil search results, got %v", m.searchResults)
	}
	if m.searchErr != nil {
		t.Errorf("expected no error, got %v", m.searchErr)
	}
}

func TestSearchError(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)

	m = sendSearchResults(m, nil, errors.New("network error"))
	if m.searchErr == nil {
		t.Error("expected search error, got nil")
	}
	if m.searchResults != nil {
		t.Errorf("expected nil results on error, got %v", m.searchResults)
	}
}

func TestSearchError_ClearedOnNewSearch(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)

	// First search errors
	m = sendSearchResults(m, nil, errors.New("network error"))
	if m.searchErr == nil {
		t.Fatal("expected search error")
	}

	// Second search succeeds
	m = sendSearchResults(m, []string{"firefox"}, nil)
	if m.searchErr != nil {
		t.Errorf("expected searchErr to be cleared, got %v", m.searchErr)
	}
	if len(m.searchResults) != 1 {
		t.Errorf("expected 1 result, got %d", len(m.searchResults))
	}
}

func TestBackendListNavigation_UnknownKey(t *testing.T) {
	m := newTestModel()
	initialCursor := m.cursor

	m = sendKey(m, "x")
	if m.cursor != initialCursor {
		t.Errorf("expected cursor unchanged, got %d", m.cursor)
	}
	if m.viewState != BackendListView {
		t.Errorf("expected view state unchanged, got %v", m.viewState)
	}
}

func TestPackageDetail_NoDoubleActionWhileLoading(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView
	m.loading = true

	// Try to install while loading — should be ignored
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m2 := updated.(Model)
	if m2.loading != true {
		t.Error("expected loading to still be true (action blocked)")
	}
}

func TestSearchView_NavigateWithoutResults(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)

	// Navigation should not cause issues when there are no results
	m = sendKey(m, "j")
	if m.searchCursor != 0 {
		t.Errorf("expected searchCursor 0, got %d", m.searchCursor)
	}

	m = sendKey(m, "k")
	if m.searchCursor != 0 {
		t.Errorf("expected searchCursor 0, got %d", m.searchCursor)
	}
}

func TestView_BackendList(t *testing.T) {
	m := newTestModel()
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_Search(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)
	view := m.View()
	if view == "" {
		t.Error("expected non-empty search view")
	}
}

func TestView_PackageDetail(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView
	view := m.View()
	if view == "" {
		t.Error("expected non-empty package detail view")
	}
}

func TestView_LoadingSpinner(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView
	m.loading = true
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view during loading")
	}
}

func TestView_ErrorState(t *testing.T) {
	m := newTestModel()
	m.selectedPkg = "firefox"
	m.viewState = PackageDetailView
	m.actionErr = errors.New("test error")
	m.statusMsg = "test of 'pkg' failed: test error"
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view on error")
	}
}

func TestView_SearchError(t *testing.T) {
	m := newTestModel()
	m = sendEnter(m)
	m.searchErr = errors.New("search failed")
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view on search error")
	}
}

func TestQuit_FromBackendList(t *testing.T) {
	m := newTestModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updated, cmd := m.Update(msg)
	m2 := updated.(Model)
	if m2.viewState != BackendListView {
		t.Errorf("expected BackendListView, got %v", m2.viewState)
	}
if cmd == nil {
		t.Errorf("expected a quit command for 'q' in BackendListView")
		} else {
			msg := cmd()
			if _, ok := msg.(tea.QuitMsg); !ok {
				t.Errorf("expected tea.QuitMsg from quit command")
			}
		}

}

func TestQuit_CtrlC(t *testing.T) {
	m := newTestModel()
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updated, cmd := m.Update(msg)
if cmd == nil {
		t.Errorf("expected a quit command for ctrl+c")
		} else {
			msg := cmd()
			if _, ok := msg.(tea.QuitMsg); !ok {
				t.Errorf("expected tea.QuitMsg from quit command")
			}
		}

	_ = updated
}

func TestWindowSize(t *testing.T) {
	m := newTestModel()
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if m2.width != 100 || m2.height != 40 {
		t.Errorf("expected width=100, height=40, got %d, %d", m2.width, m2.height)
	}
}

// --- List view tests ---

func TestListTransition_FromBackendListView(t *testing.T) {
	m := newTestModel()

	m = sendKey(m, "l")

	if m.viewState != ListView {
		t.Errorf("expected ListView, got %v", m.viewState)
	}
	if !m.loading {
		t.Error("expected loading to be true when entering list view")
	}
}

func TestListView_ReceiveResults(t *testing.T) {
	m := newTestModel()
	m = sendKey(m, "l")

	msg := listResultsMsg{
		results: []string{"firefox", "git", "linux"},
		err:     nil,
	}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	if m2.loading {
		t.Error("expected loading to be false after receiving results")
	}
	if len(m2.listResults) != 3 {
		t.Errorf("expected 3 list results, got %d", len(m2.listResults))
	}
	if m2.listCursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m2.listCursor)
	}
}

func TestListView_NavigateDown(t *testing.T) {
	m := newTestModel()
	m = sendKey(m, "l")
	msg := listResultsMsg{results: []string{"pkg1", "pkg2", "pkg3"}, err: nil}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	m2 = sendKey(m2, "j")
	if m2.listCursor != 1 {
		t.Errorf("expected cursor 1, got %d", m2.listCursor)
	}

	m2 = sendKey(m2, "down")
	if m2.listCursor != 2 {
		t.Errorf("expected cursor 2, got %d", m2.listCursor)
	}

	// Should not go past last
	m2 = sendKey(m2, "down")
	if m2.listCursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m2.listCursor)
	}
}

func TestListView_NavigateUp(t *testing.T) {
	m := newTestModel()
	m = sendKey(m, "l")
	msg := listResultsMsg{results: []string{"pkg1", "pkg2", "pkg3"}, err: nil}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	m2.listCursor = 2

	m2 = sendKey(m2, "k")
	if m2.listCursor != 1 {
		t.Errorf("expected cursor 1, got %d", m2.listCursor)
	}

	m2 = sendKey(m2, "up")
	if m2.listCursor != 0 {
		t.Errorf("expected cursor 0, got %d", m2.listCursor)
	}

	// Should not go past first
	m2 = sendKey(m2, "up")
	if m2.listCursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m2.listCursor)
	}
}

func TestListView_EmptyResults(t *testing.T) {
	m := newTestModel()
	m = sendKey(m, "l")
	msg := listResultsMsg{results: nil, err: nil}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	if m2.listResults != nil {
		t.Errorf("expected nil results, got %v", m2.listResults)
	}
	if m2.listErr != nil {
		t.Errorf("expected no error, got %v", m2.listErr)
	}
}

func TestListView_Error(t *testing.T) {
	m := newTestModel()
	m = sendKey(m, "l")
	msg := listResultsMsg{results: nil, err: errors.New("list error")}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	if m2.listErr == nil {
		t.Error("expected error, got nil")
	}
	if m2.listResults != nil {
		t.Errorf("expected nil results on error, got %v", m2.listResults)
	}
}

func TestListView_EscapeToBackendList(t *testing.T) {
	m := newTestModel()
	m = sendKey(m, "l")
	msg := listResultsMsg{results: []string{"pkg1"}, err: nil}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	m2 = sendEsc(m2)
	if m2.viewState != BackendListView {
		t.Errorf("expected BackendListView after esc, got %v", m2.viewState)
	}
	if m2.listResults != nil {
		t.Errorf("expected listResults cleared, got %v", m2.listResults)
	}
}

// --- BuildConfigView tests ---

func TestBuildConfigView_TransitionFromBackendList(t *testing.T) {
	m := newTestModel()

	m = sendKey(m, "b")

	if m.viewState != BuildConfigView {
		t.Errorf("expected BuildConfigView, got %v", m.viewState)
	}
}

func TestBuildConfigView_RendersForm(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildConfigView

	view := m.View()
	if view == "" {
		t.Error("expected non-empty build config view")
	}
	if !strings.Contains(view, "Target Directory") {
		t.Error("expected Target Directory field in form")
	}
	if !strings.Contains(view, "Kernel Version") {
		t.Error("expected Kernel Version field in form")
	}
	if !strings.Contains(view, "Skip Toolchain") {
		t.Error("expected Skip Toolchain toggle in form")
	}
	if !strings.Contains(view, "Skip Kernel") {
		t.Error("expected Skip Kernel toggle in form")
	}
}

func TestBuildConfigView_TabNavigation(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildConfigView

	// Initial field should be 0
	if m.buildConfig.currentField != 0 {
		t.Errorf("expected initial currentField 0, got %d", m.buildConfig.currentField)
	}

	// Tab to next field
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	if m2.buildConfig.currentField != 1 {
		t.Errorf("expected currentField 1 after Tab, got %d", m2.buildConfig.currentField)
	}

	// Tab multiple times to wrap around
	for i := 0; i < len(m2.buildConfig.fields)-1; i++ {
		updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyTab})
		m2 = updated.(Model)
	}
	// Should have wrapped to field 0 again
	if m2.buildConfig.currentField != 0 {
		t.Errorf("expected currentField 0 after wrap-around, got %d", m2.buildConfig.currentField)
	}

	// Verify Tab advances field correctly
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	m3 := updated.(Model)
	if m3.buildConfig.currentField != 1 {
		t.Errorf("expected currentField 1 after Tab, got %d", m3.buildConfig.currentField)
	}
}

func TestBuildConfigView_EscapeToBackendList(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildConfigView

	m = sendEsc(m)

	if m.viewState != BackendListView {
		t.Errorf("expected BackendListView after esc, got %v", m.viewState)
	}
}

func TestBuildConfigView_EnterStartsBuild(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildConfigView

	// Set required fields
	m.buildConfig.fields[1].input.SetValue("6.6")
	m.buildConfig.fields[2].input.SetValue("amd64")

	// Press Enter to start build
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)

	if m2.viewState != BuildView {
		t.Errorf("expected BuildView after Enter, got %v", m2.viewState)
	}
	if m2.buildCfg.KernelVersion != "6.6" {
		t.Errorf("expected KernelVersion 6.6, got %q", m2.buildCfg.KernelVersion)
	}
	if m2.buildCfg.Arch != "amd64" {
		t.Errorf("expected Arch amd64, got %q", m2.buildCfg.Arch)
	}
	if m2.progressChan == nil {
		t.Error("expected progressChan to be initialized")
	}
	if cmd == nil {
		t.Error("expected a command to read progress")
	}
}

func TestBuildConfigView_EnterRejectsMissingFields(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildConfigView

	// KernelVersion and Arch are empty — clear them
	m.buildConfig.fields[1].input.SetValue("")
	m.buildConfig.fields[2].input.SetValue("")

	// Press Enter — should fail validation
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)

	if m2.viewState != BuildConfigView {
		t.Errorf("expected BuildConfigView (stay on form), got %v", m2.viewState)
	}
	if m2.buildConfig.errMsg == "" {
		t.Error("expected error message on form, got empty")
	}
	if cmd != nil {
		t.Error("expected nil command when validation fails")
	}

	// Set only one field — should still fail
	m3 := newTestModel()
	m3.viewState = BuildConfigView
	m3.buildConfig.fields[1].input.SetValue("6.6")
	m3.buildConfig.fields[2].input.SetValue("")

	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := updated.(Model)

	if m4.viewState != BuildConfigView {
		t.Errorf("expected BuildConfigView when Arch is missing, got %v", m4.viewState)
	}
	if m4.buildConfig.errMsg == "" {
		t.Error("expected error message when Arch is missing")
	}
}

func TestBuildConfigView_ToggleFields(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildConfigView

	// Navigate to Skip Toolchain toggle (index 6)
	for i := 0; i < 6; i++ {
		msg := tea.KeyMsg{Type: tea.KeyTab}
		updated, _ := m.Update(msg)
		m = updated.(Model)
	}

	if m.buildConfig.currentField != 6 {
		t.Fatalf("expected currentField 6 (Skip Toolchain), got %d", m.buildConfig.currentField)
	}

	// Toggle with space
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	if !m2.buildConfig.fields[6].toggleOn {
		t.Error("expected Skip Toolchain to be toggled on")
	}

	// Toggle again
	updated, _ = m2.Update(msg)
	m3 := updated.(Model)

	if m3.buildConfig.fields[6].toggleOn {
		t.Error("expected Skip Toolchain to be toggled off")
	}
}

func TestBuildConfigView_toBuildConfig(t *testing.T) {
	b := InitialBuildConfigModel()
	b.fields[0].input.SetValue("/custom/target")
	b.fields[1].input.SetValue("6.8")
	b.fields[2].input.SetValue("arm64")
	b.fields[6].toggleOn = true

	cfg := b.toBuildConfig()

	if cfg.TargetDir != "/custom/target" {
		t.Errorf("expected TargetDir /custom/target, got %q", cfg.TargetDir)
	}
	if cfg.KernelVersion != "6.8" {
		t.Errorf("expected KernelVersion 6.8, got %q", cfg.KernelVersion)
	}
	if cfg.Arch != "arm64" {
		t.Errorf("expected Arch arm64, got %q", cfg.Arch)
	}
	if !cfg.SkipToolchain {
		t.Error("expected SkipToolchain to be true")
	}
	if cfg.Name != "AgnostikOS" {
		t.Errorf("expected default Name AgnostikOS, got %q", cfg.Name)
	}
}

// --- Build view streaming progress tests ---

func TestBuildView_ProgressStreaming(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildView
	m.progressChan = make(chan string, 20)
	m.buildMaxSteps = 14

	// Simulate receiving a progress message
	msg := progressMsg("=== Step 1/14: Prepare directories ===")
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	if len(m2.buildProgress) != 1 {
		t.Errorf("expected 1 progress message, got %d", len(m2.buildProgress))
	}
	if m2.buildCurrentStep != 1 {
		t.Errorf("expected currentStep 1, got %d", m2.buildCurrentStep)
	}
}

func TestBuildView_ProgressDisplay(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildView
	m.buildProgress = []string{
		"=== Step 1/14: Prepare directories ===",
		"=== Step 2/14: Download toolchain ===",
	}
	m.buildCurrentStep = 2
	m.buildMaxSteps = 14

	view := m.View()
	if view == "" {
		t.Error("expected non-empty build view with progress")
	}
	if !strings.Contains(view, "2/14") {
		t.Error("expected progress indicator 2/14 in view")
	}
}

func TestBuildView_CompletionSuccess(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildView
	m.progressChan = make(chan string, 20)
	m.buildCfg = manager.BuildConfig{OutputISO: "/tmp/test.iso"}

	// Receive completion
	msg := buildCompletedMsg{err: nil, iso: "/tmp/test.iso"}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	if m2.loading {
		t.Error("expected loading to be false after completion")
	}
	if !m2.buildDone {
		t.Error("expected buildDone to be true")
	}
	if m2.buildErr != nil {
		t.Errorf("expected no error, got %v", m2.buildErr)
	}
	if m2.buildOutputISO != "/tmp/test.iso" {
		t.Errorf("expected buildOutputISO /tmp/test.iso, got %q", m2.buildOutputISO)
	}
}

func TestBuildView_CompletionError(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildView
	m.progressChan = make(chan string, 20)

	msg := buildCompletedMsg{err: errors.New("build failed"), iso: ""}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)

	if m2.loading {
		t.Error("expected loading to be false after error")
	}
	if !m2.buildDone {
		t.Error("expected buildDone to be true even on error")
	}
	if m2.buildErr == nil {
		t.Error("expected error, got nil")
	}
}

func TestBuildView_SuccessResultDisplay(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildView
	m.buildDone = true
	m.buildErr = nil
	m.buildOutputISO = "/tmp/agnostikos.iso"

	view := m.View()
	if view == "" {
		t.Error("expected non-empty build view with success")
	}
	if !strings.Contains(view, "Build completed") {
		t.Error("expected success message in view")
	}
	if !strings.Contains(view, "/tmp/agnostikos.iso") {
		t.Error("expected ISO path in success view")
	}
}

func TestBuildView_ErrorResultDisplay(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildView
	m.buildDone = true
	m.buildErr = errors.New("bootstrap failed: network error")

	view := m.View()
	if view == "" {
		t.Error("expected non-empty build view with error")
	}
	if !strings.Contains(view, "Build failed") {
		t.Error("expected error message in view")
	}
	if !strings.Contains(view, "network error") {
		t.Error("expected error details in view")
	}
}

func TestBuildView_EscapeToBackendList(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildView
	m.buildDone = true
	m.buildErr = nil
	m.buildOutputISO = "/tmp/test.iso"

	m = sendEsc(m)
	if m.viewState != BackendListView {
		t.Errorf("expected BackendListView after esc, got %v", m.viewState)
	}
	if m.buildProgress != nil {
		t.Error("expected buildProgress to be cleared")
	}
}

func TestListView_ViewRendering(t *testing.T) {
	m := newTestModel()
	m = sendKey(m, "l")
	view := m.View()
	if view == "" {
		t.Error("expected non-empty list view")
	}
}

func TestBuildView_ViewRendering(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildView
	m.buildProgress = []string{"=== Step 1/14: Prepare directories ==="}
	m.buildCurrentStep = 1
	m.buildMaxSteps = 14
	view := m.View()
	if view == "" {
		t.Error("expected non-empty build view")
	}
}

func TestBuildConfigView_ViewRendering(t *testing.T) {
	m := newTestModel()
	m.viewState = BuildConfigView
	view := m.View()
	if view == "" {
		t.Error("expected non-empty build config view")
	}
}
