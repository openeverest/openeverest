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

package confirm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isTerminalTrue() bool  { return true }
func isTerminalFalse() bool { return false }

func TestGate_Yes_SkipsRegardlessOfTTY(t *testing.T) {
	t.Parallel()
	skip, err := gate(Options{Yes: true, IsTerminal: isTerminalFalse})
	assert.True(t, skip)
	assert.NoError(t, err)
}

func TestGate_JSON_IsNonInteractiveEvenOnTTY(t *testing.T) {
	t.Parallel()
	skip, err := gate(Options{JSON: true, IsTerminal: isTerminalTrue})
	assert.False(t, skip)
	assert.ErrorIs(t, err, ErrNonInteractive)
}

func TestGate_NoTTY_IsNonInteractive(t *testing.T) {
	t.Parallel()
	skip, err := gate(Options{IsTerminal: isTerminalFalse})
	assert.False(t, skip)
	assert.ErrorIs(t, err, ErrNonInteractive)
}

func TestGate_TTY_NoYes_PromptsNormally(t *testing.T) {
	t.Parallel()
	skip, err := gate(Options{IsTerminal: isTerminalTrue})
	assert.False(t, skip)
	assert.NoError(t, err)
}

func TestWillPrompt(t *testing.T) {
	t.Parallel()
	assert.False(t, WillPrompt(Options{Yes: true, IsTerminal: isTerminalTrue}))
	assert.False(t, WillPrompt(Options{JSON: true, IsTerminal: isTerminalTrue}))
	assert.False(t, WillPrompt(Options{IsTerminal: isTerminalFalse}))
	assert.True(t, WillPrompt(Options{IsTerminal: isTerminalTrue}))
}

func TestYesNo_NonInteractiveWithoutYes_ReturnsErrNonInteractive(t *testing.T) {
	t.Parallel()
	err := YesNo(context.Background(), Options{IsTerminal: isTerminalFalse}, "are you sure?")
	assert.ErrorIs(t, err, ErrNonInteractive)
}

func TestYesNo_Yes_SkipsPromptAndSucceeds(t *testing.T) {
	t.Parallel()
	err := YesNo(context.Background(), Options{Yes: true, IsTerminal: isTerminalFalse}, "are you sure?")
	assert.NoError(t, err)
}

func TestYesNo_JSON_ReturnsErrNonInteractive(t *testing.T) {
	t.Parallel()
	err := YesNo(context.Background(), Options{JSON: true, IsTerminal: isTerminalTrue}, "are you sure?")
	assert.ErrorIs(t, err, ErrNonInteractive)
}

func TestName_NonInteractiveWithoutYes_ReturnsErrNonInteractive(t *testing.T) {
	t.Parallel()
	err := Name(context.Background(), Options{IsTerminal: isTerminalFalse}, "type the name", "my-instance")
	assert.ErrorIs(t, err, ErrNonInteractive)
}

func TestName_Yes_SkipsPromptAndSucceeds(t *testing.T) {
	t.Parallel()
	err := Name(context.Background(), Options{Yes: true, IsTerminal: isTerminalFalse}, "type the name", "my-instance")
	assert.NoError(t, err)
}

func TestName_JSON_ReturnsErrNonInteractive(t *testing.T) {
	t.Parallel()
	err := Name(context.Background(), Options{JSON: true, IsTerminal: isTerminalTrue}, "type the name", "my-instance")
	assert.ErrorIs(t, err, ErrNonInteractive)
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	t.Parallel()
	require.NotErrorIs(t, ErrDeclined, ErrNameMismatch)
	require.NotErrorIs(t, ErrDeclined, ErrNonInteractive)
	require.NotErrorIs(t, ErrNameMismatch, ErrNonInteractive)
}
