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

package deletion

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

type widget struct {
	Name string
}

// captureStdout returns what fn writes to os.Stdout. Not parallel-safe
// (os.Stdout is global), so callers must not use t.Parallel.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func TestGoneCondition(t *testing.T) {
	t.Parallel()

	cond := GoneCondition("widget deleted", func(w *widget) string {
		return "still there: " + w.Name
	})

	outcome, msg := cond(nil)
	assert.Equal(t, wait.Succeeded, outcome)
	assert.Equal(t, "widget deleted", msg)

	outcome, msg = cond(&widget{Name: "foo"})
	assert.Equal(t, wait.Pending, outcome)
	assert.Equal(t, "still there: foo", msg)
}

// GonePoll's fetch/status/token-refresh classification is wait.FetchPoll's
// contract, exercised directly by pkg/cli/wait's TestFetchPoll_* suite. The
// only thing this binding decides is the NotFoundPolicy it passes through.
func TestGonePoll_Gone(t *testing.T) {
	t.Parallel()

	poll := GonePoll[widget]("widget", "my-widget", func(context.Context) (int, string, *widget, error) {
		return http.StatusNotFound, "404 Not Found", nil, nil
	})

	v, err := poll(context.Background())
	require.NoError(t, err)
	assert.Nil(t, v)
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestEmitDeleted_Pretty(t *testing.T) {
	out := captureStdout(t, func() {
		err := EmitDeleted(true, "backup storage", "my-s3", "everest")
		require.NoError(t, err)
	})
	assert.Contains(t, out, `Backup storage "my-s3" deleted`)
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestEmitDeleted_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		err := EmitDeleted(false, "instance", "my-db", "everest")
		require.NoError(t, err)
	})
	assert.JSONEq(t, `{"name":"my-db","namespace":"everest","deleted":true}`, out)
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestEmitAlreadyGone_Pretty(t *testing.T) {
	out := captureStdout(t, func() {
		err := EmitAlreadyGone(true, "instance", "my-db", "everest")
		require.NoError(t, err)
	})
	assert.Contains(t, out, `Instance "my-db" not found in namespace "everest"; nothing to delete`)
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestEmitAlreadyGone_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		err := EmitAlreadyGone(false, "backup storage", "my-s3", "everest")
		require.NoError(t, err)
	})
	assert.JSONEq(t, `{"name":"my-s3","namespace":"everest","deleted":false}`, out)
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestEmitWaitSucceeded_Pretty(t *testing.T) {
	out := captureStdout(t, func() {
		err := EmitWaitSucceeded(true, "backup storage", "my-s3", "everest")
		require.NoError(t, err)
	})
	assert.Contains(t, out, `Backup storage "my-s3" is deleted`)
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestEmitWaitSucceeded_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		err := EmitWaitSucceeded(false, "instance", "my-db", "everest")
		require.NoError(t, err)
	})
	assert.JSONEq(t, `{"name":"my-db","namespace":"everest","deleted":true}`, out)
}

func TestTitleFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"instance", "Instance"},
		{"backup storage", "Backup storage"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, titleFirst(tt.in))
	}
}

// Ensure writeResultJSON's output is exactly the wire shape callers depend
// on (field order doesn't matter for JSON, but the field set/types do).
//
//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestWriteResultJSON_Shape(t *testing.T) {
	out := captureStdout(t, func() {
		require.NoError(t, writeResultJSON("my-s3", "everest", true))
	})
	var decoded struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Deleted   bool   `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Equal(t, "my-s3", decoded.Name)
	assert.Equal(t, "everest", decoded.Namespace)
	assert.True(t, decoded.Deleted)
}
