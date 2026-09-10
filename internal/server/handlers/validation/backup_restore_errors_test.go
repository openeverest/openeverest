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

package validation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestIsValidationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "the umbrella sentinel itself", err: controller.ErrInvalidReference, want: true},
		{
			name: "a reference-validation sentinel satisfies the umbrella",
			err:  controller.ErrInstanceNotFound,
			want: true,
		},
		{
			name: "a wrapped reference-validation sentinel satisfies the umbrella",
			err:  fmt.Errorf("failed to get instance '%s': %w", "x", controller.ErrInstanceNotFound),
			want: true,
		},
		{
			name: "a newly added reference-validation sentinel satisfies the umbrella without any change here",
			err:  controller.ErrRestorePITRDateRequired,
			want: true,
		},
		{name: "unrelated error is not a validation error", err: errors.New("connection refused"), want: false},
		{
			name: "generic wrapped lookup error is not a validation error",
			err:  fmt.Errorf("failed to get backup '%s': %w", "x", errors.New("etcd unavailable")),
			want: false,
		},
		{name: "nil error is not a validation error", err: nil, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, isValidationError(tc.err))
		})
	}
}
