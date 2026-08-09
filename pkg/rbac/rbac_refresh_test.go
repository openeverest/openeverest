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

package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeInformerStarter struct {
	err error
}

func (f fakeInformerStarter) Start(context.Context, client.Object) error {
	return f.err
}

func TestStartRBACInformerPreservesStartError(t *testing.T) {
	t.Parallel()

	startErr := errors.New("informer start failed")
	err := startRBACInformer(context.Background(), fakeInformerStarter{err: startErr})
	require.Error(t, err)
	require.ErrorIs(t, err, startErr)
	assert.ErrorContains(t, err, "failed to watch RBAC ConfigMap")
}
