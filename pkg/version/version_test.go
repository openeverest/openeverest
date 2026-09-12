// everest
// Copyright (C) 2023 Percona LLC
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

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRC(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		version  string
		expected bool
	}{
		{"v0.3.0", false},
		{"v0.3.0-xx", false},
		{"v0.3.0-rc1", true},
		{"v1.3.0-rc2", true},
		{"invalid", false},
		{"", false},
	}
	for _, tc := range testCases {
		actual := IsRC(tc.version)
		assert.Equal(t, tc.expected, actual)
	}
}

func TestIsDev(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		version  string
		expected bool
	}{
		{"v0.0.0", true},
		{"v0.0.0-cf34bt", true},
		{"v0.3.0-rc1", false},
		{"v0.3.0", false},
		{"invalid", false},
		{"", false},
	}
	for _, tc := range testCases {
		actual := IsDev(tc.version)
		assert.Equal(t, tc.expected, actual)
	}
}
