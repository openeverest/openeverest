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

package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	apiManager     = "everest"
	kubectlManager = "kubectl-edit"
	// Fields owned by the API server: the spec and the actor annotations.
	apiFields = `{"f:metadata":{"f:annotations":{"f:openeverest.io/last-actor-id":{},"f:openeverest.io/last-actor-type":{}}},"f:spec":{"f:replicas":{}}}`
	// Fields owned by a kubectl edit of the spec.
	kubectlSpecFields = `{"f:spec":{"f:replicas":{}}}`
	// Fields owned by a provider that only adds its finalizer.
	providerFields = `{"f:metadata":{"f:finalizers":{}}}`
)

//nolint:gochecknoglobals // fixed reference time
var t0 = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

func entry(manager, fields string, at time.Duration) metav1.ManagedFieldsEntry {
	ts := metav1.NewTime(t0.Add(at))
	return metav1.ManagedFieldsEntry{
		Manager: manager, Operation: metav1.ManagedFieldsOperationUpdate, Time: &ts,
		FieldsType: "FieldsV1", FieldsV1: metav1.NewFieldsV1(fields),
	}
}

func object(annotations map[string]string, entries ...metav1.ManagedFieldsEntry) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{Name: "db1", Annotations: annotations, ManagedFields: entries}
}

//nolint:gochecknoglobals // shared fixture
var aliceAnnotations = map[string]string{AnnotationActorType: "user", AnnotationActorID: "alice"}

func TestCreatorActor(t *testing.T) {
	t.Parallel()

	tcases := []struct {
		desc string
		obj  metav1.Object
		want Actor
	}{
		{
			desc: "objects created through the API carry their creator",
			obj:  object(aliceAnnotations, entry(apiManager, apiFields, 0)),
			want: Actor{Type: "user", ID: "alice"},
		},
		{
			desc: "objects created in Kubernetes are attributed to the oldest field manager",
			obj:  object(nil, entry("provider", providerFields, time.Minute), entry("kubectl-create", kubectlSpecFields, 0)),
			want: Actor{Type: ActorTypeKubernetesClient, ID: "kubectl-create"},
		},
		{
			desc: "nothing to attribute",
			obj:  object(nil),
			want: Actor{},
		},
	}
	for _, tc := range tcases {
		assert.Equal(t, tc.want, CreatorActor(tc.obj), tc.desc)
	}
}

func TestUpdaterActor(t *testing.T) {
	t.Parallel()

	tcases := []struct {
		desc string
		obj  metav1.Object
		want Actor
	}{
		{
			desc: "a spec change through the API is attributed to the stamped user",
			obj:  object(aliceAnnotations, entry(apiManager, apiFields, time.Minute)),
			want: Actor{Type: "user", ID: "alice"},
		},
		{
			desc: "a later kubectl edit isn't attributed to the stale annotations of an earlier API call",
			obj: object(aliceAnnotations,
				entry(apiManager, apiFields, 0),
				entry(kubectlManager, kubectlSpecFields, time.Minute),
			),
			want: Actor{Type: ActorTypeKubernetesClient, ID: kubectlManager},
		},
		{
			desc: "a later metadata-only write by a controller doesn't change who updated the spec",
			obj: object(aliceAnnotations,
				entry(apiManager, apiFields, time.Minute),
				entry("provider", providerFields, 2*time.Minute),
			),
			want: Actor{Type: "user", ID: "alice"},
		},
	}
	for _, tc := range tcases {
		assert.Equal(t, tc.want, UpdaterActor(tc.obj), tc.desc)
	}
}

func TestDeleterActor(t *testing.T) {
	t.Parallel()

	deleting := func(deletedAt time.Duration, entries ...metav1.ManagedFieldsEntry) *metav1.ObjectMeta {
		obj := object(aliceAnnotations, entries...)
		ts := metav1.NewTime(t0.Add(deletedAt))
		obj.DeletionTimestamp = &ts
		return obj
	}

	tcases := []struct {
		desc string
		obj  metav1.Object
		want Actor
	}{
		{
			desc: "the API stamps the deleting user right before deleting",
			obj:  deleting(time.Hour+time.Second, entry(apiManager, apiFields, time.Hour)),
			want: Actor{Type: "user", ID: "alice"},
		},
		{
			desc: "an old stamp doesn't attribute a later deletion",
			obj:  deleting(2*time.Hour, entry(apiManager, apiFields, time.Hour)),
			want: Actor{},
		},
		{
			desc: "a deletion right after a kubectl write is not attributed to the API stamp",
			obj: deleting(time.Hour,
				entry(apiManager, apiFields, 0),
				entry(kubectlManager, kubectlSpecFields, time.Hour),
			),
			want: Actor{},
		},
		{
			desc: "the object is not being deleted",
			obj:  object(aliceAnnotations),
			want: Actor{},
		},
	}
	for _, tc := range tcases {
		assert.Equal(t, tc.want, DeleterActor(tc.obj), tc.desc)
	}
}
