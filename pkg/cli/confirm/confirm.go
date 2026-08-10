// everest
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

// Package confirm handles the --yes / --json / TTY check and prompt shared
// by every delete command.
package confirm

import (
	"context"
	"errors"
	"os"

	"golang.org/x/term"

	"github.com/openeverest/openeverest/v2/pkg/cli/tui"
)

var (
	// ErrDeclined means the user said no.
	ErrDeclined = errors.New("action was not confirmed")
	// ErrNameMismatch means the typed name didn't match.
	ErrNameMismatch = errors.New("typed name did not match; nothing was deleted")
	// ErrNonInteractive means no safe way to prompt (no TTY, or --json) and --yes wasn't set.
	ErrNonInteractive = errors.New("confirmation required in non-interactive mode; re-run with --yes")
)

// Options for a confirmation check.
type Options struct {
	Yes  bool
	JSON bool

	// IsTerminal overrides the TTY check; nil uses the real one. Set in tests.
	IsTerminal func() bool
}

func (o Options) isTerminal() bool {
	if o.IsTerminal != nil {
		return o.IsTerminal()
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// WillPrompt reports whether a prompt will actually be shown.
func WillPrompt(o Options) bool {
	if o.Yes {
		return false
	}
	return !o.JSON && o.isTerminal()
}

// gate checks --yes / --json / TTY. True means skip the prompt; an error
// means don't prompt at all.
func gate(o Options) (bool, error) {
	if o.Yes {
		return true, nil
	}
	if o.JSON || !o.isTerminal() {
		return false, ErrNonInteractive
	}
	return false, nil
}

// YesNo asks a y/N question. Returns nil if confirmed or skipped via --yes.
func YesNo(ctx context.Context, o Options, message string) error {
	skip, err := gate(o)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	confirmed, err := tui.NewConfirm(ctx, message).Run()
	if err != nil {
		return err
	}
	if !confirmed {
		return ErrDeclined
	}
	return nil
}

// Name asks the user to type wantName back verbatim.
func Name(ctx context.Context, o Options, message, wantName string) error {
	skip, err := gate(o)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	validate := func(typed string) error {
		if typed != wantName {
			return ErrNameMismatch
		}
		return nil
	}

	_, err = tui.NewInput(ctx, message, tui.WithInputValidation(validate)).Run()
	if errors.Is(err, tui.ErrUserInterrupted) {
		return ErrDeclined
	}
	return err
}
