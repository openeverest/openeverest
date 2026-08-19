// everest
// Copyright (C) 2025 Percona LLC
// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tui provides UI elements for the CLI.
package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//nolint:gochecknoglobals // immutable TUI style/keybinding definitions
var (
	// ------
	// Style applied to option label in case its checkbox is selected.
	optionStyle = lipgloss.NewStyle().
			Foreground(
			lipgloss.AdaptiveColor{Light: "#0E5FB5", Dark: "#93C7FF"},
		)
	// Style applied to option label in case its checkbox is unselected.
	hoverOptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#0B4A8C", Dark: "#62AEFF"}).
				Background(lipgloss.AdaptiveColor{Light: "#E2EFFC", Dark: "#2F435B"})

	// Selected checkbox symbol.
	selectedCheckBox = "[X]"

	// Unselected checkbox symbol.
	unselectedCheckbox = "[ ]"

	// cursor current position symbol.
	cursorSymbol = ">"

	// ----------
	// Key bindings.
	multiSelectKeys = multiSelectKeyMap{
		Confirm: confirmKeyBinding,
		Quit:    quitKeyBinding,
		Up: key.NewBinding(
			key.WithKeys(tea.KeyUp.String()),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys(tea.KeyDown.String()),
			key.WithHelp("↓", "down"),
		),
		Space: key.NewBinding(
			key.WithKeys(tea.KeySpace.String()),
			key.WithHelp("space", "select/unselect"),
		),
	}
)

type (
	// multiSelectKeyMap defines a set of keybindings. To work for help it must satisfy
	// key.Map. It could also very easily be a map[string]key.Binding.
	multiSelectKeyMap struct {
		Confirm key.Binding
		Quit    key.Binding
		Up      key.Binding
		Down    key.Binding
		Space   key.Binding
	}

	// MultiSelectOption represents an option in the multi-select list.
	MultiSelectOption struct {
		Text     string
		Selected bool
	}

	// MultiSelect represents a multi-select list.
	MultiSelect struct {
		keys      multiSelectKeyMap
		help      help.Model
		Message   string              // message to display
		Choices   []MultiSelectOption // possible options user may choose from
		cursor    int                 // which item in choice list our cursor is pointing at
		p         *tea.Program
		done      bool // user has confirmed the selection
		interrupt bool // set in case user wants to quit (Esc or Ctrl+c)
	}
)

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k multiSelectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Up, k.Down, k.Space, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k multiSelectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.Up, k.Down, k.Space, k.Quit}, // first column
		{}, // second column
	}
}

// NewMultiSelect creates a new multi-select list.
func NewMultiSelect(ctx context.Context, message string, choices []MultiSelectOption) MultiSelect {
	m := MultiSelect{
		keys:    multiSelectKeys,
		help:    newHelpModel(),
		Message: message,
		Choices: choices,
	}

	p := tea.NewProgram(m, tea.WithContext(ctx))
	m.p = p
	return m
}

// Run starts the multi-select list.
// It returns the selected options and error.
func (m MultiSelect) Run() ([]MultiSelectOption, error) {
	model, err := m.p.Run()
	result, ok := model.(MultiSelect)
	if !ok {
		return nil, fmt.Errorf("unexpected model type: %T", model)
	}
	if result.interrupt {
		os.Exit(1)
	}
	return result.Choices, err
}

// Init initializes the multi-select list.
// Implements bubbletea.Model interface.
func (m MultiSelect) Init() tea.Cmd {
	// we do not need command here.
	return nil
}

// Update updates the multi-select list.
// Implements bubbletea.Model interface.
func (m MultiSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keys.Confirm):
			m.done = true
			return m, tea.Quit
		case key.Matches(keyMsg, m.keys.Quit):
			m.interrupt = true
			m.done = true
			return m, tea.Quit

		// The "up" key move the cursor up
		case key.Matches(keyMsg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" key move the cursor down
		case key.Matches(keyMsg, m.keys.Down):
			if m.cursor < len(m.Choices)-1 {
				m.cursor++
			}

		// The spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case key.Matches(keyMsg, m.keys.Space):
			m.Choices[m.cursor].Selected = !m.Choices[m.cursor].Selected
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

// View renders the multi-select list.
// It returns a string representation of the UI.
// Implements bubbletea.Model interface.
func (m MultiSelect) View() string {
	// The header
	s := strings.Builder{}
	fmt.Fprintf(&s, "%s %s\n", failureStyle.Render("❓"), textStyle.Render(m.Message))

	// Iterate over our choices
	for i, choice := range m.Choices {
		// Render the row
		s.WriteString(drawLine(m.cursor == i, choice.Selected, choice.Text))
	}

	if !m.done {
		fmt.Fprintf(&s, "\n%s\n", m.help.View(m.keys))
	}

	// Send the UI for rendering
	return s.String()
}

func drawLine(hover, checked bool, label string) string {
	// Template contains: <option selected/unselected mark> <option label>
	// Examples:
	// [X] MySQL
	// [ ] PostgreSQL
	const tmpl = " %s %s %s"
	mark := selectedCheckBox
	if !checked {
		mark = unselectedCheckbox
	}

	if hover {
		return fmt.Sprintf("%s\n", hoverOptionStyle.Render(fmt.Sprintf(tmpl, cursorSymbol, mark, label)))
	}
	return fmt.Sprintf("%s\n", optionStyle.Render(fmt.Sprintf(tmpl, " ", mark, label)))
}
