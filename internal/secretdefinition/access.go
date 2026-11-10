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

package secretdefinition

import (
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

// ErrManagedOutside is returned by Writable for a Secret that was not created
// through the OpenEverest API: Git, a secret manager or whoever applied it
// owns its data, and would revert a change made here.
var ErrManagedOutside = errors.New("managed outside OpenEverest; change it at its source")

// ManagedByControllerError is returned by Writable for a Secret with a
// controller owner reference, such as an ExternalSecret or a SealedSecret:
// the controller would revert a change to the Secret, or recreate it.
type ManagedByControllerError struct {
	// Kind is the kind of the controller owner.
	Kind string
	// Name is the name of the controller owner.
	Name string
}

func (e *ManagedByControllerError) Error() string {
	return fmt.Sprintf("managed by %s/%s", e.Kind, e.Name)
}

// Labelled reports whether labels name a definition: exactly one source label
// and openeverest.io/definition, or the core definition ca-bundle and no
// source label. Whoever wrote it, the HTTP API, kubectl, Git or a secret
// manager, a labelled Secret is one OpenEverest can use, so the HTTP API shows
// it.
func Labelled(labels map[string]string) bool {
	_, err := RefFromLabels(labels)
	return err == nil
}

// Tracked reports whether labels mark a Secret core tracks: a labelled Secret
// that no Provider defines. Those are the Secrets the typed fields of core
// kinds reference, such as a MonitoringDestination's credentials and CA.
// Provider-defined Secrets are referenced from Instance parameters, which core
// does not follow yet.
func Tracked(labels map[string]string) bool {
	ref, err := RefFromLabels(labels)
	return err == nil && ref.Kind != KindProvider
}

// Writable reports whether the HTTP API may change the data of secret. It may
// only if the API created the Secret (openeverest.io/managed: "true") and no
// controller owns it. It returns a *ManagedByControllerError for a Secret with
// a controller owner reference, whether or not the API created it, and
// ErrManagedOutside for any other Secret the API did not create.
func Writable(secret metav1.Object) error {
	if owner := metav1.GetControllerOfNoCopy(secret); owner != nil {
		return &ManagedByControllerError{Kind: owner.Kind, Name: owner.Name}
	}
	if secret.GetLabels()[common.OpenEverestManagedLabel] != "true" {
		return ErrManagedOutside
	}
	return nil
}
