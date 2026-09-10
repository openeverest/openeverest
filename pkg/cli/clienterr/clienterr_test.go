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

package clienterr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
)

func TestMessage(t *testing.T) {
	t.Parallel()
	badRequest := "bad request"
	msg, ok := clienterr.Message(&client.Error{Message: &badRequest})
	require.True(t, ok)
	assert.Equal(t, badRequest, msg)

	_, ok = clienterr.Message(nil, &client.Error{})
	assert.False(t, ok)

	first, second := "first", "second"
	msg, ok = clienterr.Message(nil, &client.Error{Message: &first}, &client.Error{Message: &second})
	require.True(t, ok)
	assert.Equal(t, first, msg)
}
