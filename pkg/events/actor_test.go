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

	alice := Actor{Type: "user", ID: "alice"}
	assert.Equal(t, alice, CreatorActor(object(aliceAnnotations, entry(apiManager, apiFields, 0))),
		"objects created through the API carry their creator")

	assert.Equal(t, Actor{Type: ActorTypeKubernetesClient, ID: "kubectl-create"},
		CreatorActor(object(nil, entry("provider", providerFields, time.Minute), entry("kubectl-create", kubectlSpecFields, 0))),
		"objects created in Kubernetes are attributed to the oldest field manager")

	assert.Equal(t, Actor{}, CreatorActor(object(nil)), "nothing to attribute")
}

func TestUpdaterActor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Actor{Type: "user", ID: "alice"},
		UpdaterActor(object(aliceAnnotations, entry(apiManager, apiFields, time.Minute))),
		"a spec change through the API is attributed to the stamped user")

	assert.Equal(t, Actor{Type: ActorTypeKubernetesClient, ID: kubectlManager},
		UpdaterActor(object(aliceAnnotations,
			entry(apiManager, apiFields, 0),
			entry(kubectlManager, kubectlSpecFields, time.Minute),
		)),
		"a later kubectl edit isn't attributed to the stale annotations of an earlier API call")

	assert.Equal(t, Actor{Type: "user", ID: "alice"},
		UpdaterActor(object(aliceAnnotations,
			entry(apiManager, apiFields, time.Minute),
			entry("provider", providerFields, 2*time.Minute),
		)),
		"a later metadata-only write by a controller doesn't change who updated the spec")
}

func TestDeleterActor(t *testing.T) {
	t.Parallel()

	deleting := func(deletedAt time.Duration, entries ...metav1.ManagedFieldsEntry) *metav1.ObjectMeta {
		obj := object(aliceAnnotations, entries...)
		ts := metav1.NewTime(t0.Add(deletedAt))
		obj.DeletionTimestamp = &ts
		return obj
	}

	assert.Equal(t, Actor{Type: "user", ID: "alice"},
		DeleterActor(deleting(time.Hour+time.Second, entry(apiManager, apiFields, time.Hour))),
		"the API stamps the deleting user right before deleting")

	assert.Equal(t, Actor{}, DeleterActor(deleting(2*time.Hour, entry(apiManager, apiFields, time.Hour))),
		"an old stamp doesn't attribute a later deletion")

	assert.Equal(t, Actor{}, DeleterActor(deleting(time.Hour,
		entry(apiManager, apiFields, 0),
		entry(kubectlManager, kubectlSpecFields, time.Hour),
	)), "a deletion right after a kubectl write is not attributed to the API stamp")

	assert.Equal(t, Actor{}, DeleterActor(object(aliceAnnotations)), "the object is not being deleted")
}
