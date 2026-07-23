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

package instance

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

func TestCreateExitCode(t *testing.T) {
	// Not parallel: createExitCode reads/prints via package-level createOpts.
	createOpts.Name = "my-db"

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, 0},
		{"failed phase", &wait.FailedError{Message: "entered the Failed phase"}, exitFailed},
		{"generic error", errors.New("boom"), exitFailed},
		{"timeout", fmt.Errorf("wrap: %w", wait.ErrTimeout), exitTimeout},
		{"cancelled", fmt.Errorf("wrap: %w", context.Canceled), exitCanceled},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, createExitCode(tc.err))
		})
	}
}

func TestValidateWaitFlags(t *testing.T) {
	// Not parallel: mutates the shared createCmd flag state and createOpts.
	reset := func(wait bool, timeoutChanged bool) {
		createOpts.Wait = wait
		createOpts.Timeout = 10_000_000_000 // 10s, positive
		_ = createCmd.Flags().Set("timeout", "10s")
		if !timeoutChanged {
			// Reset the flag's Changed bit to model "user didn't pass --timeout".
			createCmd.Flags().Lookup("timeout").Changed = false
		}
	}

	t.Run("timeout without wait errors", func(t *testing.T) {
		reset(false, true)
		err := validateWaitFlags(createCmd)
		assert.ErrorContains(t, err, "only valid together with --wait")
	})

	t.Run("wait with non-positive timeout errors", func(t *testing.T) {
		reset(true, false)
		createOpts.Timeout = 0
		err := validateWaitFlags(createCmd)
		assert.ErrorContains(t, err, "positive duration")
	})

	t.Run("wait with default timeout is fine", func(t *testing.T) {
		reset(true, false)
		assert.NoError(t, validateWaitFlags(createCmd))
	})
}
