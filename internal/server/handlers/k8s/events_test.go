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

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestFilterNamespaces(t *testing.T) {
	t.Parallel()

	source := watch.NewFake()
	w := filterNamespaces(source, []string{"team-a", "team-b"})
	defer w.Stop()

	go func() {
		for _, ns := range []string{"team-a", "team-c", "team-b"} {
			source.Add(&eventsv1.Event{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: ns + "-event"}})
		}
		// A bookmark is an empty Event without a namespace; it must not be dropped.
		source.Action(watch.Bookmark, &eventsv1.Event{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "7"}})
	}()

	var got []string
	for ev := range w.ResultChan() {
		obj := ev.Object.(*eventsv1.Event) //nolint:forcetypeassert // only events are sent
		if ev.Type == watch.Bookmark {
			assert.Equal(t, "7", obj.ResourceVersion)
			break
		}
		got = append(got, obj.Name)
	}
	assert.Equal(t, []string{"team-a-event", "team-b-event"}, got)
}
