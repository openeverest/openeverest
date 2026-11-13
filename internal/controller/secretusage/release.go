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

package secretusage

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// listPageSize bounds the Secrets one list call of Release returns.
const listPageSize = 500

// Release takes the Reconciler's marks off every Secret: the finalizer and
// the count keys of refs. The controller runs it when Secret protection is
// disabled, and registers no Reconciler then, so nothing puts them back.
//
// Secrets are listed metadata only, a page at a time, labelled or not,
// since a Secret whose labels were removed may still carry the finalizer;
// c should read the API server, not a cache. Each Secret carrying a mark
// is patched with a lock on its resourceVersion, so a concurrent change to
// its finalizers or annotations is never overwritten, and on a conflict it
// is read again and retried. Release goes on past a Secret it fails to
// release, reports every failure, and returns how many Secrets it changed.
// A Secret already being deleted goes once its last finalizer is removed,
// as whoever deleted it asked.
func Release(ctx context.Context, c client.Client, refs []Referrer) (int, error) {
	keys := make([]string, 0, len(refs))
	for _, ref := range refs {
		keys = append(keys, ref.CountKey)
	}
	var errs []error
	released := 0
	continueToken := ""
	for {
		list := &metav1.PartialObjectMetadataList{}
		list.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("SecretList"))
		if err := c.List(ctx, list, client.Limit(listPageSize), client.Continue(continueToken)); err != nil {
			return released, errors.Join(append(errs, fmt.Errorf("list Secrets: %w", err))...)
		}
		for i := range list.Items {
			secret := &list.Items[i]
			if !carriesMarks(secret, keys) {
				continue
			}
			changed, err := unmark(ctx, c, secret, keys)
			if err != nil {
				errs = append(errs, fmt.Errorf("release Secret %s: %w", client.ObjectKeyFromObject(secret), err))
				continue
			}
			if changed {
				released++
			}
		}
		continueToken = list.Continue
		if continueToken == "" {
			return released, errors.Join(errs...)
		}
	}
}

// carriesMarks reports whether secret carries the finalizer or one of keys.
func carriesMarks(secret client.Object, keys []string) bool {
	if controllerutil.ContainsFinalizer(secret, finalizer) {
		return true
	}
	for _, key := range keys {
		if _, ok := secret.GetAnnotations()[key]; ok {
			return true
		}
	}
	return false
}

// unmark removes the finalizer and keys from secret, reading it again and
// retrying when the patch conflicts. It reports whether it patched secret;
// a Secret that is gone needs nothing.
func unmark(ctx context.Context, c client.Client, secret *metav1.PartialObjectMetadata, keys []string) (bool, error) {
	key := client.ObjectKeyFromObject(secret)
	secret = secret.DeepCopy()
	secret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	changed := false
	first := true
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if !first {
			if err := c.Get(ctx, key, secret); err != nil {
				return err
			}
		}
		first = false
		base := secret.DeepCopy()
		removed := controllerutil.RemoveFinalizer(secret, finalizer)
		annotations := secret.GetAnnotations()
		for _, k := range keys {
			if _, ok := annotations[k]; ok {
				delete(annotations, k)
				removed = true
			}
		}
		if !removed {
			return nil
		}
		if err := c.Patch(ctx, secret, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{})); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return changed, err
}
