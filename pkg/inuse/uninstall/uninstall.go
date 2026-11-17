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

// Package uninstall checks whether core can be uninstalled. Core protects
// objects in use with a finalizer that only its controller removes, once
// nothing uses them any more. Uninstalling core while it still holds one
// would leave the object, and the namespace holding it, impossible to
// delete. So the uninstall is refused until core has released them all,
// the way an operator's resources are deleted before the operator.
//
// Check only reads. It is shared by the controller's uninstall-check
// subcommand, which the chart's pre-delete hook runs, and by everestctl
// uninstall.
package uninstall

import (
	"context"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// DocsURL documents uninstalling OpenEverest, removing the finalizers by
// hand included.
const DocsURL = "https://openeverest.io/documentation/current/uninstall/uninstall_everest_helm.html"

// listPageSize bounds the objects one list call returns.
const listPageSize = 500

// maxNamed bounds the objects the error names per kind.
const maxNamed = 10

// Protected is a kind core protects with an in-use finalizer.
type Protected struct {
	// Kind is the kind of the objects. Check lists them, metadata only, in
	// every namespace.
	Kind schema.GroupVersionKind
	// Finalizer is the finalizer core's controller holds on them.
	Finalizer string
}

// Registry lists the kinds Check looks at. Add an entry when core starts
// protecting another kind: a finalizer missing here lets core be
// uninstalled while it still holds it.
func Registry() []Protected {
	return []Protected{
		{
			Kind:      monitoringv1alpha1.GroupVersion.WithKind("MonitoringDestination"),
			Finalizer: monitoringv1alpha1.InUseProtectionFinalizer,
		},
		{
			// The backup controller's backupStorageFinalizer; a test in its
			// package keeps the two equal.
			Kind:      backupv1alpha1.GroupVersion.WithKind("BackupStorage"),
			Finalizer: "backupstorage.backup.openeverest.io/in-use-protection",
		},
		{
			Kind:      corev1.SchemeGroupVersion.WithKind("Secret"),
			Finalizer: common.OpenEverestInUseProtectionFinalizer,
		},
	}
}

// Object is an object core still protects.
type Object struct {
	Namespace string
	Name      string
	// Deleting is set once its deletion has started: core releases it as
	// soon as nothing uses it any more.
	Deleting bool
}

func (o Object) String() string {
	name := o.Name
	if o.Namespace != "" {
		name = o.Namespace + "/" + o.Name
	}
	if o.Deleting {
		name += " (being deleted)"
	}
	return name
}

// Held lists the objects of one kind that core still protects.
type Held struct {
	Kind    string
	Objects []Object
}

// HeldError is the error Check returns while core still protects objects.
type HeldError struct {
	// Held lists them per kind, in Registry order, leaving out kinds with
	// none.
	Held []Held
}

func (e *HeldError) Error() string {
	total := 0
	for _, h := range e.Held {
		total += len(h.Objects)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "OpenEverest still protects %d %s with finalizers that only its controller removes; "+
		"uninstalling it now would leave them, and the namespaces holding them, impossible to delete:\n",
		total, plural(total, "object"),
	)
	for _, h := range e.Held {
		fmt.Fprintf(&b, "  %d %s: %s\n", len(h.Objects), plural(len(h.Objects), h.Kind), summarise(h.Objects))
	}
	b.WriteString("Delete these objects, or the namespaces holding them, while OpenEverest is still running, " +
		"so that it releases them, then uninstall again. " +
		"`everestctl uninstall` does this for the database namespaces. " +
		"An object being deleted is released once nothing uses it any more.\n")
	fmt.Fprintf(&b, "To uninstall anyway, run `helm uninstall --no-hooks`, then remove the finalizers by hand: %s", DocsURL)
	return b.String()
}

// AnyDeleting reports whether an object held is already being deleted. If
// none is, waiting releases nothing: only deleting objects, or the
// namespaces holding them, does. A Secret is released once the objects
// using it are gone, and those carry a finalizer too, so they show here.
func (e *HeldError) AnyDeleting() bool {
	for _, h := range e.Held {
		for _, o := range h.Objects {
			if o.Deleting {
				return true
			}
		}
	}
	return false
}

// Check returns a *HeldError naming the objects core still protects with an
// in-use finalizer, or nil when there is none. A kind that is not served,
// e.g. whose CRD is not installed, holds nothing. Check never writes.
func Check(ctx context.Context, r client.Reader) error {
	held := &HeldError{}
	for _, p := range Registry() {
		objects, err := protected(ctx, r, p)
		if err != nil {
			return fmt.Errorf("list %s objects: %w", p.Kind.Kind, err)
		}
		if len(objects) > 0 {
			held.Held = append(held.Held, Held{Kind: p.Kind.Kind, Objects: objects})
		}
	}
	if len(held.Held) == 0 {
		return nil
	}
	return held
}

// protected lists the objects of p.Kind carrying p.Finalizer, sorted.
func protected(ctx context.Context, r client.Reader, p Protected) ([]Object, error) {
	var objects []Object
	continueToken := ""
	for {
		list := &metav1.PartialObjectMetadataList{}
		list.SetGroupVersionKind(p.Kind.GroupVersion().WithKind(p.Kind.Kind + "List"))
		if err := r.List(ctx, list, client.Limit(listPageSize), client.Continue(continueToken)); err != nil {
			if meta.IsNoMatchError(err) {
				return nil, nil
			}
			return nil, err
		}
		for i := range list.Items {
			item := &list.Items[i]
			if controllerutil.ContainsFinalizer(item, p.Finalizer) {
				objects = append(objects, Object{
					Namespace: item.Namespace,
					Name:      item.Name,
					Deleting:  !item.DeletionTimestamp.IsZero(),
				})
			}
		}
		continueToken = list.Continue
		if continueToken == "" {
			break
		}
	}
	slices.SortFunc(objects, func(a, b Object) int {
		return strings.Compare(a.Namespace+"/"+a.Name, b.Namespace+"/"+b.Name)
	})
	return objects, nil
}

// summarise names at most maxNamed objects.
func summarise(objects []Object) string {
	names := make([]string, 0, min(len(objects), maxNamed))
	for _, o := range objects[:min(len(objects), maxNamed)] {
		names = append(names, o.String())
	}
	if len(objects) > maxNamed {
		return fmt.Sprintf("%s and %d more", strings.Join(names, ", "), len(objects)-maxNamed)
	}
	return strings.Join(names, ", ")
}

func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}
