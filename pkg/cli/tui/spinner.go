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
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
	"golang.org/x/term"
)

const (
	spinnerInterval = 150 * time.Millisecond
)

//nolint:gochecknoglobals // immutable TUI style/keybinding definitions
var (
	currentStepTitleStyle = helperTextStyle.Bold(false)
	successStepTitleStyle = successStyle
	failedStepTitleStyle  = failureStyle
	successMark           = successStyle.Render("✅")
	failureMark           = failureStyle.Render("❌") // x
)

type (
	// Step provides a way to run a function with a
	// pretty loading spinner animation.
	Step struct {
		// Desc is a human-readable description of the step.
		Desc string
		// F is the function that will be called to execute the step.
		F func(ctx context.Context) error
	}

	// spinnerKeyMap defines a set of keybindings. To work for help it must satisfy
	// key.Map. It could also very easily be a map[string]key.Binding.
	spinnerKeyMap struct {
		Quit key.Binding
	}

	// Spinner represents action with spinner element.
	Spinner struct {
		steps     []Step
		index     int
		keys      spinnerKeyMap
		help      help.Model
		spinner   spinner.Model
		p         *tea.Program
		interrupt bool               // set in case user wants to quit (Esc or Ctrl+c)
		done      bool               // all steps have been completed
		stepError error              // error occurred during step execution
		ctx       context.Context    //nolint:containedctx // bubbletea models receive no context in Init/Update; the element's run context must be carried on the model
		l         *zap.SugaredLogger // logger
	}

	// SpinnerOption is used to set options when initializing Spinner.
	// Spinner can accept a variable number of options.
	//
	// Example usage:
	//
	//	p := NewSpinner(ctx, WithPretty(bool))
	SpinnerOption func(m *Spinner)
)

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k spinnerKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k spinnerKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit}, // first column
		{},       // second column
	}
}

// NewSpinner creates a new spinner element.
func NewSpinner(ctx context.Context, l *zap.SugaredLogger, steps []Step, opts ...SpinnerOption) Spinner {
	s := spinner.New(
		spinner.WithSpinner(spinner.Points),
		spinner.WithStyle(helperTextStyle),
	)
	m := Spinner{
		spinner: s,
		steps:   steps,
		keys: spinnerKeyMap{
			Quit: quitKeyBinding,
		},

		help: newHelpModel(),
		ctx:  ctx,
		l:    l,
	}

	teaOpts := []tea.ProgramOption{tea.WithContext(ctx)}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// If we're not running in a terminal, we can't read input.
		// Input is used for reading user's keys for Help model (quit key).
		teaOpts = append(teaOpts, tea.WithInput(nil))
	}

	p := tea.NewProgram(m, teaOpts...)
	m.p = p
	// Apply all options to the program.
	// must be done after m.p=p because some options may need to access m.p
	for _, opt := range opts {
		opt(&m)
	}

	return m
}

// WithSpinnerPrettyPrint lets you specify pretty print for the spinner.
// It will print the spinner in a pretty way.
// In case the spinner is not pretty, it will print the spinner in a verbose way.
func WithSpinnerPrettyPrint(prettyPrint bool) SpinnerOption {
	return func(m *Spinner) {
		if !prettyPrint {
			tea.WithoutRenderer()(m.p)
		}
	}
}

// Run starts the spinner element.
// It returns the selected options and error.
func (m Spinner) Run() error {
	model, err := m.p.Run()
	if err != nil {
		return err
	}

	result, ok := model.(Spinner)
	if !ok {
		return fmt.Errorf("unexpected model type: %T", model)
	}

	if result.interrupt {
		os.Exit(1)
	}

	return result.stepError
}

// Init initializes the spinner element.
// Implements bubbletea.Model interface.
func (m Spinner) Init() tea.Cmd {
	// Run spinner and first step
	return tea.Batch(m.spinner.Tick, runStep(m.ctx, m.l, m.steps[m.index]))
}

// Update updates the spinner element.
// Implements bubbletea.Model interface.
func (m Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			m.interrupt = true
			m.done = true
			return m, tea.Quit
		}
	case errorStepMsg:
		// An error occurred during the step execution.
		stp := m.steps[m.index]
		m.stepError = msg
		m.done = true

		return m, tea.Sequence(
			tea.Printf("%s  %s", failureMark, failedStepTitleStyle.Render(stp.Desc)), // mark step as failed.
			tea.Quit, // exit the element
		)
	case executedStepMsg:
		stp := m.steps[m.index]
		if m.index >= len(m.steps)-1 {
			// All steps have been executed. We're done!
			m.done = true
			return m, tea.Sequence(
				tea.Printf("%s  %s", successMark, successStepTitleStyle.Render(stp.Desc)), // mark step as successfully.
				tea.Quit, // exit the element
			)
		}

		// Run next step.
		m.index++
		return m, tea.Sequence(
			tea.Printf("%s  %s", successMark, successStepTitleStyle.Render(stp.Desc)), // print success message above our program
			runStep(m.ctx, m.l, m.steps[m.index]),                                     // run the next step
		)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the spinner element.
// It returns a string representation of the UI.
// Implements bubbletea.Model interface.
func (m Spinner) View() string {
	if m.done {
		// All steps have been executed.
		// No need to print anymore.
		return ""
	}

	// Print intermediate state.
	s := strings.Builder{}
	// spin
	fmt.Fprintf(&s, "%s ", m.spinner.View())
	// step title info
	s.WriteString(currentStepTitleStyle.Render(m.steps[m.index].Desc))
	// help info
	fmt.Fprintf(&s, "\n\n%s\n", m.help.View(m.keys))

	return s.String()
}

type (
	// executedStepMsg is a message that is sent when a step is executed.
	executedStepMsg string

	// errorStepMsg is a message that is sent when a step returns an error.
	errorStepMsg error
)

// Wrapper function to run step in a separate goroutine.
func runStep(ctx context.Context, l *zap.SugaredLogger, step Step) tea.Cmd {
	return tea.Tick(spinnerInterval, func(_ time.Time) tea.Msg {
		l.Debug("Running step: ", step.Desc)
		if err := step.F(ctx); err != nil {
			l.Errorw("Error occurred during step execution", "step", step.Desc, "error", err)
			return errorStepMsg(err)
		}

		return executedStepMsg(step.Desc)
	})
}
