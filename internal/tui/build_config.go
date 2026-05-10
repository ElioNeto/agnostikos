package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ElioNeto/agnostikos/internal/manager"
)

// buildConfigField represents a form field in the BuildConfigView.
type buildConfigField struct {
	label    string
	input    textinput.Model
	toggle   bool   // when true, field is a boolean toggle
	toggleOn bool   // current toggle state
	key      string // field key for building the config
}

// BuildConfigViewModel holds the form state for configuring a build.
type BuildConfigViewModel struct {
	fields       []buildConfigField
	currentField int
	focused      bool
	errMsg       string // validation error message, shown on the form
}

// InitialBuildConfigModel creates the default build config form model.
func InitialBuildConfigModel() BuildConfigViewModel {
	makeInput := func(placeholder, value string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.SetValue(value)
		ti.CharLimit = 200
		ti.Width = 50
		return ti
	}

	fields := []buildConfigField{
		{label: "Target Directory", input: makeInput("/mnt/data/agnostikOS/rootfs", ""), key: "targetDir"},
		{label: "Kernel Version", input: makeInput("6.6", "6.6"), key: "kernelVersion"},
		{label: "Architecture", input: makeInput("amd64", ""), key: "arch"},
		{label: "Output ISO Path", input: makeInput("/mnt/data/agnostikOS/build/agnostikos-latest.iso", ""), key: "outputISO"},
		{label: "Busybox Version", input: makeInput("1.36.1", "1.36.1"), key: "busyboxVersion"},
		{label: "Jobs (parallelism)", input: makeInput("4", "4"), key: "jobs"},
		{label: "Skip Toolchain", toggle: true, key: "skipToolchain"},
		{label: "Skip Kernel Build", toggle: true, key: "skipKernel"},
	}

	// Focus the first text input
	for i := range fields {
		if !fields[i].toggle {
			fields[i].input.Blur()
		}
	}
	if len(fields) > 0 && !fields[0].toggle {
		fields[0].input.Focus()
	}

	return BuildConfigViewModel{
		fields:       fields,
		currentField: 0,
		focused:      true,
	}
}

// toBuildConfig extracts a manager.BuildConfig from the form fields.
func (b BuildConfigViewModel) toBuildConfig() manager.BuildConfig {
	cfg := manager.BuildConfig{}
	for _, f := range b.fields {
		switch f.key {
		case "targetDir":
			cfg.TargetDir = f.input.Value()
		case "kernelVersion":
			cfg.KernelVersion = f.input.Value()
		case "arch":
			cfg.Arch = f.input.Value()
		case "outputISO":
			cfg.OutputISO = f.input.Value()
		case "busyboxVersion":
			cfg.BusyboxVersion = f.input.Value()
		case "jobs":
			cfg.Jobs = f.input.Value()
		case "skipToolchain":
			cfg.SkipToolchain = f.toggleOn
		case "skipKernel":
			cfg.SkipKernel = f.toggleOn
		}
	}
	if cfg.Name == "" {
		cfg.Name = "AgnostikOS"
	}
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}
	return cfg
}

// Update processes key messages for the build config form.
func (b BuildConfigViewModel) Update(msg tea.Msg) (BuildConfigViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			b.currentField = (b.currentField + 1) % len(b.fields)
			b.updateFocus()
			return b, nil
		case "shift+tab":
			b.currentField = (b.currentField - 1 + len(b.fields)) % len(b.fields)
			b.updateFocus()
			return b, nil
		case " ":
			// Toggle boolean fields with space
			if b.fields[b.currentField].toggle {
				b.fields[b.currentField].toggleOn = !b.fields[b.currentField].toggleOn
				return b, nil
			}
		}
	}

	// Forward key to the current text input field
	if !b.fields[b.currentField].toggle {
		var cmd tea.Cmd
		b.fields[b.currentField].input, cmd = b.fields[b.currentField].input.Update(msg)
		return b, cmd
	}

	return b, nil
}

// updateFocus sets focus on the current field's input and blurs others.
func (b *BuildConfigViewModel) updateFocus() {
	for i := range b.fields {
		if b.fields[i].toggle {
			continue
		}
		if i == b.currentField {
			b.fields[i].input.Focus()
		} else {
			b.fields[i].input.Blur()
		}
	}
}

// View renders the build config form.
func (b BuildConfigViewModel) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Build AgnosticOS ISO — Configuration"))
	s.WriteString("\n\n")

	for i, f := range b.fields {
		prefix := "  "
		if i == b.currentField {
			prefix = selectedStyle.Render("> ")
		}

		if f.toggle {
			// Boolean toggle
			check := "[ ]"
			if f.toggleOn {
				check = "[x]"
			}
			if i == b.currentField {
				check = selectedStyle.Render(check)
			}
			fmt.Fprintf(&s, "%s%s %s\n", prefix, check, f.label)
		} else {
			label := lipgloss.NewStyle().Width(20).Render(f.label)
			fmt.Fprintf(&s, "%s%s %s\n", prefix, label, f.input.View())
		}
	}

	if b.errMsg != "" {
		s.WriteString("\n")
		s.WriteString(errorStyle.Render("❌ " + b.errMsg))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Tab: next field  •  Shift+Tab: previous field  •  Space: toggle  •  Enter: start build  •  Esc: back"))
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("q: quit"))

	return s.String()
}

// reset clears all form fields to defaults.
func (b *BuildConfigViewModel) reset() {
	*b = InitialBuildConfigModel()
}
