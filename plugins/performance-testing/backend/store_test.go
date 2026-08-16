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

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_SaveGet(t *testing.T) {
	t.Parallel()

	s := newMemoryStore()
	run := Run{ID: "run-1", Instance: "pg-poc", Status: RunStatusRunning, StartedAt: time.Now()}
	require.NoError(t, s.Save(run))

	got, ok := s.Get("run-1")
	require.True(t, ok)
	assert.Equal(t, run.Instance, got.Instance)
}

func TestMemoryStore_Get_MissingReturnsFalse(t *testing.T) {
	t.Parallel()

	s := newMemoryStore()
	_, ok := s.Get("does-not-exist")
	assert.False(t, ok)
}

func TestMemoryStore_List_MostRecentFirst(t *testing.T) {
	t.Parallel()

	s := newMemoryStore()
	now := time.Now()
	require.NoError(t, s.Save(Run{ID: "older", StartedAt: now.Add(-time.Hour)}))
	require.NoError(t, s.Save(Run{ID: "newer", StartedAt: now}))

	list := s.List()
	require.Len(t, list, 2)
	assert.Equal(t, "newer", list[0].ID)
	assert.Equal(t, "older", list[1].ID)
}

func TestMemoryStore_Save_EmptyIDRejected(t *testing.T) {
	t.Parallel()

	s := newMemoryStore()
	err := s.Save(Run{ID: ""})
	require.Error(t, err)
}
