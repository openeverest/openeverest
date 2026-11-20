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

package reconciler

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/validation"
)

// reasonValidationWarning is the reason of the Event that carries validation
// warnings.
const reasonValidationWarning = "ValidationWarning"

// validate runs the validation pipeline on the Instance and records its
// warnings. The error is for a Provider that cannot be read.
func (r *ProviderReconciler) validate(ctx context.Context, in *v1alpha1.Instance) (field.ErrorList, error) {
	provider := &v1alpha1.Provider{}
	if err := r.Get(ctx, client.ObjectKey{Name: in.Spec.ProviderRef.Name}, provider); err != nil {
		return nil, fmt.Errorf("fetching provider for validation: %w", err)
	}
	res := validation.Run(ctx, r.provider, controller.ValidationInput{Instance: in.DeepCopy(), Provider: provider})
	r.recordWarnings(in, res.Warnings)
	return res.Errors, nil
}

// recordWarnings records the warnings as one Warning Event per generation.
func (r *ProviderReconciler) recordWarnings(in *v1alpha1.Instance, warnings []string) {
	if len(warnings) == 0 || r.recorder == nil {
		return
	}
	if last, ok := r.warned.Load(in.UID); ok && last == in.Generation {
		return
	}
	r.recorder.Eventf(in, nil, corev1.EventTypeWarning, reasonValidationWarning, "Validate", "%s", strings.Join(warnings, "; "))
	r.warned.Store(in.UID, in.Generation)
}
