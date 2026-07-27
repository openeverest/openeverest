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

package backup

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

func TestCreateExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, 0},
		{"failed state", &wait.FailedError{Message: "entered the Failed state"}, exitFailed},
		{"generic error", errors.New("boom"), exitFailed},
		{"timeout", fmt.Errorf("wrap: %w", wait.ErrTimeout), exitTimeout},
		{"cancelled", fmt.Errorf("wrap: %w", context.Canceled), exitCanceled},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, createExitCode(tc.err, 15*time.Minute))
		})
	}
}

func TestValidateWaitFlags(t *testing.T) {
	t.Parallel()

	t.Run("timeout without wait errors", func(t *testing.T) {
		t.Parallel()
		err := validateWaitFlags(false, true, 10*time.Second)
		assert.ErrorContains(t, err, "only valid together with --wait")
	})

	t.Run("wait with non-positive timeout errors", func(t *testing.T) {
		t.Parallel()
		err := validateWaitFlags(true, false, 0)
		assert.ErrorContains(t, err, "positive duration")
	})

	t.Run("wait without --timeout flag is fine", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, validateWaitFlags(true, false, 10*time.Second))
	})
}
