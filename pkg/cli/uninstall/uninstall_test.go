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

package uninstall

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUninstallHasNoClusterTypeState is a regression guard for issue #2888:
// the uninstall flow used to run Kubernetes environment detection (behind an
// inverted --skip-env-detection check) whose result, u.clusterType, was never
// consumed anywhere in the package. Both the detection call and the field
// that stored its result have been removed; this asserts the field does not
// silently reappear.
func TestUninstallHasNoClusterTypeState(t *testing.T) {
	t.Parallel()

	_, ok := reflect.TypeFor[Uninstall]().FieldByName("clusterType")
	assert.False(t, ok, "Uninstall must not carry an unused clusterType field")
}

// TestConfigSkipEnvDetectionRemainsSettable ensures Config.SkipEnvDetection
// is kept for backward compatibility with existing --skip-env-detection
// invocations, even though the uninstall flow no longer reads it.
func TestConfigSkipEnvDetectionRemainsSettable(t *testing.T) {
	t.Parallel()

	cfg := Config{SkipEnvDetection: true}
	assert.True(t, cfg.SkipEnvDetection)

	cfg = Config{SkipEnvDetection: false}
	assert.False(t, cfg.SkipEnvDetection)
}
