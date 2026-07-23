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
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	instancecli "github.com/openeverest/openeverest/v2/pkg/cli/instance"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

func TestCreateExitCode(t *testing.T) {
	t.Parallel()

	opts := &instancecli.CreateOptions{Name: "my-db", Timeout: time.Minute}

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
			t.Parallel()
			assert.Equal(t, tc.want, createExitCode(tc.err, opts, true))
		})
	}
}

func TestValidateWaitFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		wait           bool
		timeout        time.Duration
		timeoutChanged bool   // whether --timeout was explicitly passed
		wantErr        string // substring; empty means no error expected
	}{
		{"timeout without wait", false, 10 * time.Second, true, "only valid together with --wait"},
		{"wait with non-positive timeout", true, 0, false, "positive duration"},
		{"wait with default timeout", true, 10 * time.Minute, false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{}
			cmd.Flags().Duration(cli.FlagInstanceTimeout, 10*time.Minute, "")
			if tc.timeoutChanged {
				require.NoError(t, cmd.Flags().Set(cli.FlagInstanceTimeout, "10s"))
			}
			opts := &instancecli.CreateOptions{Wait: tc.wait, Timeout: tc.timeout}

			err := validateWaitFlags(cmd, opts)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}
