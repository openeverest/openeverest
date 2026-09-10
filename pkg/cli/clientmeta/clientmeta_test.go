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

package clientmeta_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/pkg/cli/clientmeta"
)

// object mimics a generated client model: a struct with a *metav1.ObjectMeta
// field and no methods.
type object struct {
	Metadata *metav1.ObjectMeta
}

func withAge(name string, age time.Duration) object {
	return object{Metadata: &metav1.ObjectMeta{
		Name:              name,
		CreationTimestamp: metav1.NewTime(time.Now().Add(-age).UTC()),
	}}
}

func withoutTimestamp(name string) object {
	return object{Metadata: &metav1.ObjectMeta{Name: name}}
}

func names(items []object) []string {
	out := make([]string, len(items))
	for i, o := range items {
		if o.Metadata == nil {
			out[i] = "<nil>"
			continue
		}
		out[i] = o.Metadata.Name
	}
	return out
}

func TestSortByRecency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []object
		want  []string
	}{
		{
			name: "newest first, missing timestamps last",
			items: []object{
				withAge("oldest", 48*time.Hour),
				withAge("newest", time.Hour),
				withoutTimestamp("no-timestamp"),
				withAge("middle", 24*time.Hour),
			},
			want: []string{"newest", "middle", "oldest", "no-timestamp"},
		},
		{
			name: "name tiebreak when all timestamps missing",
			items: []object{
				withoutTimestamp("b"),
				withoutTimestamp("a"),
				withoutTimestamp("c"),
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "nil metadata sorts last with missing-timestamp group",
			items: []object{
				{Metadata: nil},
				withAge("dated", time.Hour),
			},
			want: []string{"dated", "<nil>"},
		},
		{
			name:  "empty slice is a no-op",
			items: []object{},
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			clientmeta.SortByRecency(tc.items, func(o *object) *metav1.ObjectMeta { return o.Metadata })
			assert.Equal(t, tc.want, names(tc.items))
		})
	}
}

func TestSortByRecency_EqualTimestampsFallBackToName(t *testing.T) {
	t.Parallel()

	ts := metav1.NewTime(time.Now().UTC())
	items := []object{
		{Metadata: &metav1.ObjectMeta{Name: "b", CreationTimestamp: ts}},
		{Metadata: &metav1.ObjectMeta{Name: "a", CreationTimestamp: ts}},
	}

	clientmeta.SortByRecency(items, func(o *object) *metav1.ObjectMeta { return o.Metadata })
	assert.Equal(t, []string{"a", "b"}, names(items))
}
