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

package waitcmd

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, 0},
		{"failed state", &wait.FailedError{Message: "entered a Failed state"}, ExitFailed},
		{"generic error", errors.New("boom"), ExitFailed},
		{"timeout", fmt.Errorf("wrap: %w", wait.ErrTimeout), ExitTimeout},
		{"cancelled", fmt.Errorf("wrap: %w", context.Canceled), ExitCanceled},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ExitCode(tc.err, zap.NewNop().Sugar(), true, "cancelled", "timed out")
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidateWaitFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		waitSet        bool
		timeout        time.Duration
		timeoutChanged bool // whether --timeout was explicitly passed
		wantErr        string
	}{
		{"timeout without wait", false, 10 * time.Second, true, "only valid together with --wait"},
		{"wait with non-positive timeout", true, 0, false, "positive duration"},
		{"wait with default timeout", true, 10 * time.Minute, false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateWaitFlags(tc.waitSet, tc.timeoutChanged, tc.timeout)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}
