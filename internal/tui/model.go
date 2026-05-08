package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ElioNeto/agnostikos/internal/manager"
)

// ViewState represents which screen the TUI is currently showing.
type ViewState int

const (
	BackendListView ViewState = iota
	SearchView
	PackageDetailView
)

// Style definitions for the TUI.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#44FF44"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)
)

// searchResultsMsg carries async search results back to the update loop.
type searchResultsMsg struct {
	results []string
	err     error
}

// actionCompletedMsg carries async install/remove results.
type actionCompletedMsg struct {
	action string // "install" or "remove"
	pkg    string
	err    error
}

// Model is the main Bubble Tea model implementing tea.Model.
type Model struct {
	manager *manager.AgnosticManager

	// View state
	viewState ViewState
	backends  []string
	cursor    int

	// Search
	searchInput   textinput.Model
	searchResults []string
	searchCursor  int

	// Package detail
	selectedPkg string

	// Async operations
	spinner   spinner.Model
	loading   bool
	statusMsg string
	actionErr error
	searchErr error

	// Terminal dimensions
	width  int
	height int
}

// InitialModel creates a new Model with default state.
func InitialModel(mgr *manager.AgnosticManager) Model {
	backends := mgr.ListBackends()

	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
	s.Spinner = spinner.Dot

	ti := textinput.New()
	ti.Placeholder = "Type a package name and press Enter..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return Model{
		manager:     mgr,
		backends:    backends,
		viewState:   BackendListView,
		cursor:      0,
		searchInput: ti,
		spinner:     s,
	}
}

// Init initializes the Bubble Tea program.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global quit on 'q' or ctrl+c (except in text input)
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "q" && m.viewState != SearchView {
			return m, tea.Quit
		}

		switch m.viewState {
		case BackendListView:
			return m.handleBackendListKey(msg)
		case SearchView:
			return m.handleSearchViewKey(msg)
		case PackageDetailView:
			return m.handlePackageDetailKey(msg)
		}

	case searchResultsMsg:
		m.loading = false
		if msg.err != nil {
			m.searchErr = msg.err
			m.searchResults = nil
		} else {
			m.searchErr = nil
			m.searchResults = msg.results
			m.searchCursor = 0
		}
		return m, nil

	case actionCompletedMsg:
		m.loading = false
		if msg.err != nil {
			m.actionErr = msg.err
			m.statusMsg = fmt.Sprintf("%s of '%s' failed: %s",
				msg.action, msg.pkg, msg.err)
		} else {
			m.actionErr = nil
			m.statusMsg = fmt.Sprintf("%s of '%s' completed successfully",
				msg.action, msg.pkg)
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// handleBackendListKey processes key presses on the backend list screen.
func (m Model) handleBackendListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.backends)-1 {
			m.cursor++
		}
	case "enter":
		m.viewState = SearchView
		m.searchInput.Focus()
		m.searchInput.SetValue("")
		m.searchResults = nil
		m.searchErr = nil
		m.searchCursor = 0
		return m, textinput.Blink
	}
	return m, nil
}

// handleSearchViewKey processes key presses on the search screen.
func (m Model) handleSearchViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Navigation keys (must be handled before textinput update)
	switch msg.String() {
	case "up", "k":
		if len(m.searchResults) > 0 && m.searchCursor > 0 {
			m.searchCursor--
			return m, nil
		}
	case "down", "j":
		if len(m.searchResults) > 0 && m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
			return m, nil
		}
	case "enter":
		query := m.searchInput.Value()
		if len(m.searchResults) > 0 {
			// Results exist — select the highlighted result
			m.selectedPkg = m.searchResults[m.searchCursor]
			m.viewState = PackageDetailView
			m.statusMsg = ""
			m.actionErr = nil
			return m, nil
		}
		if query != "" {
			// No results yet — trigger a new search
			m.loading = true
			m.searchErr = nil
			m.searchResults = nil
			return m, m.searchCmd(query)
		}
	case "esc":
		// Go back to backend list
		m.viewState = BackendListView
		return m, nil
	}

	// Update text input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// handlePackageDetailKey processes key presses on the package detail screen.
func (m Model) handlePackageDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "i":
		if !m.loading {
			m.loading = true
			m.actionErr = nil
			m.statusMsg = ""
			return m, m.installCmd(m.selectedPkg)
		}
	case "r":
		if !m.loading {
			m.loading = true
			m.actionErr = nil
			m.statusMsg = ""
			return m, m.removeCmd(m.selectedPkg)
		}
	case "esc":
		// Go back to search
		m.viewState = SearchView
		return m, nil
	}
	return m, nil
}

// searchCmd returns a tea.Cmd that performs a package search asynchronously.
func (m Model) searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		backend := m.backends[m.cursor]
		svc := m.manager.Backends[backend]
		results, err := svc.Search(query)
		return searchResultsMsg{results: results, err: err}
	}
}

// installCmd returns a tea.Cmd that installs a package asynchronously.
func (m Model) installCmd(pkg string) tea.Cmd {
	return func() tea.Msg {
		backend := m.backends[m.cursor]
		svc := m.manager.Backends[backend]
		err := svc.Install(pkg)
		return actionCompletedMsg{action: "install", pkg: pkg, err: err}
	}
}

// removeCmd returns a tea.Cmd that removes a package asynchronously.
func (m Model) removeCmd(pkg string) tea.Cmd {
	return func() tea.Msg {
		backend := m.backends[m.cursor]
		svc := m.manager.Backends[backend]
		err := svc.Remove(pkg)
		return actionCompletedMsg{action: "remove", pkg: pkg, err: err}
	}
}

// View renders the current screen based on viewState.
func (m Model) View() string {
	switch m.viewState {
	case BackendListView:
		return m.backendListView()
	case SearchView:
		return m.searchView()
	case PackageDetailView:
		return m.packageDetailView()
	default:
		return "Unknown view state"
	}
}

// backendListView renders the backend selection screen.
func (m Model) backendListView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("AgnostikOS - Package Manager"))
	b.WriteString("\n\n")
	b.WriteString("Select a backend:\n\n")

	for i, backend := range m.backends {
		cursor := "  "
		line := fmt.Sprintf("%s%s", cursor, backend)
		if i == m.cursor {
			line = selectedStyle.Render(fmt.Sprintf("> %s", backend))
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • q: quit"))

	return b.String()
}

// searchView renders the search screen with input and results.
func (m Model) searchView() string {
	var b strings.Builder
	backend := m.backends[m.cursor]
	b.WriteString(titleStyle.Render(fmt.Sprintf("Searching in: %s", backend)))
	b.WriteString("\n\n")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n")

	if m.loading {
		b.WriteString(fmt.Sprintf("\n  %s Searching...\n", m.spinner.View()))
	} else if m.searchErr != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("\nError: %s\n", m.searchErr)))
	} else if len(m.searchResults) > 0 {
		b.WriteString(fmt.Sprintf("\nResults (%d):\n\n", len(m.searchResults)))
		for i, result := range m.searchResults {
			line := fmt.Sprintf("  %s", result)
			if i == m.searchCursor {
				line = selectedStyle.Render(fmt.Sprintf("> %s", result))
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	} else if m.searchInput.Value() != "" {
		b.WriteString("\nPress Enter to search...\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • enter: search/select • esc: back • q: quit"))

	return b.String()
}

// packageDetailView renders the package detail with install/remove actions.
func (m Model) packageDetailView() string {
	var b strings.Builder
	backend := m.backends[m.cursor]
	b.WriteString(titleStyle.Render("Package Detail"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Package: %s\n", m.selectedPkg))
	b.WriteString(fmt.Sprintf("Backend: %s\n", backend))
	b.WriteString("\n")

	if m.loading {
		b.WriteString(fmt.Sprintf("\n  %s Working...\n", m.spinner.View()))
	} else if m.actionErr != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("%s\n\n", m.statusMsg)))
	} else if m.statusMsg != "" {
		b.WriteString(successStyle.Render(fmt.Sprintf("%s\n\n", m.statusMsg)))
	}

	b.WriteString("[i] Install  [r] Remove\n")
	b.WriteString(helpStyle.Render("esc: back • q: quit"))

	return b.String()
}
