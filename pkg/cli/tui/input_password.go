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
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type (
	// inputPasswordKeyMap defines a set of keybindings. To work for help it must satisfy
	// key.Map. It could also very easily be a map[string]key.Binding.
	inputPasswordKeyMap struct {
		Confirm key.Binding
		Quit    key.Binding
	}

	// InputPassword represents a password input element.
	InputPassword struct {
		keys         inputPasswordKeyMap
		help         help.Model
		textInput    textinput.Model
		p            *tea.Program
		done         bool              // user has confirmed the input
		interrupt    bool              // set in case user wants to quit (Esc or Ctrl+c)
		validateFunc ValidateInputFunc // function to validate the input
		hint         string            // hint to show in the input field
	}

	// InputPasswordOption is used to set options when initializing InputPassword.
	// InputPassword can accept a variable number of options.
	//
	// Example usage:
	//
	//	p := NewInputPassword(ctx, WithValidation(validationFunc), WithHint(hintMessage))
	InputPasswordOption func(m *InputPassword)
)

// Key bindings.
var inputPasswordKeys = inputPasswordKeyMap{ //nolint:gochecknoglobals // immutable TUI style/keybinding definitions
	Confirm: confirmKeyBinding,
	Quit:    quitKeyBinding,
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k inputPasswordKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k inputPasswordKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.Quit}, // first column
		{},                  // second column
	}
}

// NewInputPassword creates a new password input element.
func NewInputPassword(ctx context.Context, message string, opts ...InputPasswordOption) InputPassword {
	ti := textinput.New()
	ti.Prompt = fmt.Sprintf("❓ %s: ", message)
	ti.PromptStyle = textStyle
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'
	ti.Cursor.Style = textStyle
	ti.Cursor.TextStyle = textStyle
	ti.Focus()

	m := InputPassword{
		keys:      inputPasswordKeys,
		help:      newHelpModel(),
		textInput: ti,
	}

	// Apply all options to the program.
	for _, opt := range opts {
		opt(&m)
	}

	p := tea.NewProgram(m, tea.WithContext(ctx))
	m.p = p
	return m
}

// WithPasswordHint lets you specify a hint message that will be shown in the dialog
// when the user is prompted for input.
func WithPasswordHint(hint string) InputPasswordOption {
	return func(m *InputPassword) {
		if hint != "" {
			m.hint = fmt.Sprintf("HINT: %s", hint)
		}
	}
}

// WithPasswordValidation lets you specify a validate function that will be
// used to validate user's input.
func WithPasswordValidation(validateFunc ValidateInputFunc) InputPasswordOption {
	return func(m *InputPassword) {
		if validateFunc != nil {
			m.validateFunc = validateFunc
		}
	}
}

// Run runs the password input element.
func (m InputPassword) Run() (string, error) {
	model, err := m.p.Run()
	if err != nil {
		return "", err
	}

	input, ok := model.(InputPassword)
	if !ok {
		return "", fmt.Errorf("unexpected model type: %T", model)
	}

	if input.interrupt {
		return "", ErrUserInterrupted
	}

	if input.validateFunc != nil {
		if err := input.validateFunc(input.textInput.Value()); err != nil {
			return "", err
		}
	}
	return input.textInput.Value(), nil
}

// Init initializes the password input element.
// Implements bubbletea.Model interface.
func (m InputPassword) Init() tea.Cmd {
	return textinput.Blink
}

// Update updates the password input element.
// Implements bubbletea.Model interface.
func (m InputPassword) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keys.Confirm):
			m.done = true
			m.textInput.Blur()
			cmd = tea.Quit
		case key.Matches(keyMsg, m.keys.Quit):
			m.done = true
			m.interrupt = true
			m.textInput.Blur()
			cmd = tea.Quit
		}
	}

	var textInputCmd tea.Cmd
	m.textInput, textInputCmd = m.textInput.Update(msg)
	return m, tea.Sequence(textInputCmd, cmd)
}

// View renders the password element view.
// Implements bubbletea.Model interface.
func (m InputPassword) View() string {
	s := strings.Builder{}
	fmt.Fprintf(&s, "%s\n", m.textInput.View())

	if !m.done {
		if m.hint != "" {
			fmt.Fprintf(&s, "\n%s\n", helperTextStyle.Render(m.hint))
		}
		fmt.Fprintf(&s, "\n%s\n", m.help.View(m.keys))
	}

	return s.String()
}
