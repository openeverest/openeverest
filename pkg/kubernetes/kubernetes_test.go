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

// Package kubernetes provides functionality for kubernetes.
package kubernetes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	eventsv1 "k8s.io/api/events/v1"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMergeNamesspacesEnvVar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ns1  string
		ns2  string
		want string
	}{
		{
			name: "0-0",
			ns1:  "",
			ns2:  "",
			want: "",
		},
		{
			name: "1-0",
			ns1:  "ns1",
			ns2:  "",
			want: "ns1",
		},
		{
			name: "0-1",
			ns1:  "",
			ns2:  "ns2",
			want: "ns2",
		},
		{
			name: "1-1",
			ns1:  "ns1",
			ns2:  "ns2",
			want: "ns1,ns2",
		},
		{
			name: "1-2",
			ns1:  "ns1",
			ns2:  "ns2,ns3",
			want: "ns1,ns2,ns3",
		},
		{
			name: "1-2 unsorted",
			ns1:  "ns2",
			ns2:  "ns3,ns1",
			want: "ns1,ns2,ns3",
		},
		{
			name: "1-2 extra commas",
			ns1:  ",ns1,",
			ns2:  ",,ns2,,,,ns3,,,",
			want: "ns1,ns2,ns3",
		},
		{
			name: "2-2",
			ns1:  "ns4,ns1",
			ns2:  "ns2,ns3",
			want: "ns1,ns2,ns3,ns4",
		},
		{
			name: "0-2",
			ns1:  "",
			ns2:  "ns2,ns3",
			want: "ns2,ns3",
		},
		{
			name: "2-0 unsorted",
			ns1:  "ns4,ns1",
			ns2:  "",
			want: "ns1,ns4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mergedNS := mergeNamespacesEnvVar(tt.ns1, tt.ns2)
			assert.Equal(t, tt.want, mergedNS)
		})
	}
}

func TestWatchLifecycleEventsForwardsBookmarks(t *testing.T) {
	t.Parallel()

	queries := make(chan map[string][]string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			http.NotFound(w, r)
			return
		}
		queries <- r.URL.Query()
		w.Header().Set("Content-Type", "application/json;stream=watch")
		_, err := io.WriteString(w, `{"type":"BOOKMARK","object":{"apiVersion":"events.k8s.io/v1","kind":"Event","metadata":{"resourceVersion":"45"}}}`+"\n")
		assert.NoError(t, err)
	}))
	defer server.Close()

	groupVersion := eventsv1.SchemeGroupVersion
	mapper := apiMeta.NewDefaultRESTMapper([]schema.GroupVersion{groupVersion})
	mapper.AddSpecific(groupVersion.WithKind("Event"), groupVersion.WithResource("events"), groupVersion.WithResource("events"), apiMeta.RESTScopeNamespace)
	mapper.AddSpecific(groupVersion.WithKind("EventList"), groupVersion.WithResource("events"), groupVersion.WithResource("events"), apiMeta.RESTScopeNamespace)
	watchClient, err := ctrlclient.NewWithWatch(&rest.Config{Host: server.URL}, ctrlclient.Options{Scheme: CreateScheme(), Mapper: mapper})
	require.NoError(t, err)
	events, err := (&Kubernetes{watchClient: watchClient}).WatchLifecycleEvents(context.Background(), "", "40")
	require.NoError(t, err)
	defer events.Stop()

	select {
	case query := <-queries:
		assert.Equal(t, []string{"true"}, query["allowWatchBookmarks"])
		assert.Equal(t, []string{"40"}, query["resourceVersion"])
	case <-time.After(5 * time.Second):
		t.Fatal("watch request was not received")
	}
	select {
	case event := <-events.ResultChan():
		assert.Equal(t, watch.Bookmark, event.Type)
		assert.Equal(t, "45", event.Object.(*eventsv1.Event).ResourceVersion)
	case <-time.After(5 * time.Second):
		t.Fatal("watch did not forward the bookmark")
	}
}
