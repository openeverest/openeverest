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
	"errors"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	// ValidateInputFunc is a function that returns an error if the input is invalid.
	ValidateInputFunc func(string) error
)

//nolint:gochecknoglobals // immutable TUI style/keybinding definitions
var (
	// Common errors.

	// ErrUserInterrupted is returned when the user interrupts an input element (Esc or Ctrl+c).
	ErrUserInterrupted = errors.New("user interrupted")

	// Common styles.

	// Style is applied to the prompt text.
	textStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(
			lipgloss.AdaptiveColor{Light: "#000000", Dark: "ffffff"},
		)

	// Style is applied to the successful result.
	successStyle = lipgloss.NewStyle().
			Foreground(
			lipgloss.AdaptiveColor{Light: "#000000", Dark: "#5fd700"},
		)

	// Style is applied to the failure result.
	failureStyle = lipgloss.NewStyle().
			Foreground(
			lipgloss.AdaptiveColor{Light: "#B10810", Dark: "#F37C6F"},
		)

	// Style is applied to the helper text: supported key combinations, etc.
	helperTextStyle = lipgloss.NewStyle().
			Foreground(
			lipgloss.AdaptiveColor{Light: "#1A7362", Dark: "#30D1B2"},
		)

	// Common key bindings.

	// key binding and help description for Confirm action.
	confirmKeyBinding = key.NewBinding(
		key.WithKeys(tea.KeyEnter.String()),
		key.WithHelp("Enter", "confirm"),
	)

	// key binding and help description for Quit action.
	quitKeyBinding = key.NewBinding(
		key.WithKeys(tea.KeyEsc.String(), tea.KeyCtrlC.String()),
		key.WithHelp("Esc/Ctrl+c", "quit"),
	)
)

func newHelpModel() help.Model {
	model := help.New()
	model.ShortSeparator = " | "
	model.Styles.ShortKey = textStyle
	model.Styles.ShortDesc = helperTextStyle
	model.Styles.ShortSeparator = helperTextStyle
	return model
}
