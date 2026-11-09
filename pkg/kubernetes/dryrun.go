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

package kubernetes

import (
	"context"
	"slices"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type dryRunKey struct{}

// WithDryRun returns a copy of ctx whose Kubernetes writes are sent with dryRun=All, so none is persisted.
func WithDryRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, dryRunKey{}, true)
}

// IsDryRun reports whether ctx was returned by WithDryRun.
func IsDryRun(ctx context.Context) bool {
	dryRun, _ := ctx.Value(dryRunKey{}).(bool)
	return dryRun
}

// withDryRun wraps c so that every write made with a WithDryRun context carries dryRun=All.
func withDryRun(c ctrlclient.WithWatch) ctrlclient.WithWatch { //nolint:ireturn // the interceptor is only reachable through the client interface
	return interceptor.NewClient(c, interceptor.Funcs{
		Create: func(ctx context.Context, c ctrlclient.WithWatch, obj ctrlclient.Object, opts ...ctrlclient.CreateOption) error {
			return c.Create(ctx, obj, dryRunOpts(ctx, opts)...)
		},
		Update: func(ctx context.Context, c ctrlclient.WithWatch, obj ctrlclient.Object, opts ...ctrlclient.UpdateOption) error {
			return c.Update(ctx, obj, dryRunOpts(ctx, opts)...)
		},
		Patch: func(ctx context.Context, c ctrlclient.WithWatch, obj ctrlclient.Object, patch ctrlclient.Patch, opts ...ctrlclient.PatchOption) error {
			return c.Patch(ctx, obj, patch, dryRunOpts(ctx, opts)...)
		},
		Apply: func(ctx context.Context, c ctrlclient.WithWatch, obj runtime.ApplyConfiguration, opts ...ctrlclient.ApplyOption) error {
			return c.Apply(ctx, obj, dryRunOpts(ctx, opts)...)
		},
		Delete: func(ctx context.Context, c ctrlclient.WithWatch, obj ctrlclient.Object, opts ...ctrlclient.DeleteOption) error {
			return c.Delete(ctx, obj, dryRunOpts(ctx, opts)...)
		},
		DeleteAllOf: func(ctx context.Context, c ctrlclient.WithWatch, obj ctrlclient.Object, opts ...ctrlclient.DeleteAllOfOption) error {
			return c.DeleteAllOf(ctx, obj, dryRunOpts(ctx, opts)...)
		},
		SubResourceCreate: func(ctx context.Context, c ctrlclient.Client, subResource string, obj, subResourceObj ctrlclient.Object, opts ...ctrlclient.SubResourceCreateOption) error {
			return c.SubResource(subResource).Create(ctx, obj, subResourceObj, dryRunOpts(ctx, opts)...)
		},
		SubResourceUpdate: func(ctx context.Context, c ctrlclient.Client, subResource string, obj ctrlclient.Object, opts ...ctrlclient.SubResourceUpdateOption) error {
			return c.SubResource(subResource).Update(ctx, obj, dryRunOpts(ctx, opts)...)
		},
		SubResourcePatch: func(ctx context.Context, c ctrlclient.Client, subResource string, obj ctrlclient.Object, patch ctrlclient.Patch, opts ...ctrlclient.SubResourcePatchOption) error {
			return c.SubResource(subResource).Patch(ctx, obj, patch, dryRunOpts(ctx, opts)...)
		},
		SubResourceApply: func(ctx context.Context, c ctrlclient.Client, subResource string, obj runtime.ApplyConfiguration, opts ...ctrlclient.SubResourceApplyOption) error {
			return c.SubResource(subResource).Apply(ctx, obj, dryRunOpts(ctx, opts)...)
		},
	})
}

// dryRunOpts appends DryRunAll to opts when ctx asks for a dry run.
func dryRunOpts[O any](ctx context.Context, opts []O) []O {
	if !IsDryRun(ctx) {
		return opts
	}
	return append(slices.Clip(opts), any(ctrlclient.DryRunAll).(O)) //nolint:forcetypeassert // every O is an option interface DryRunAll implements
}
