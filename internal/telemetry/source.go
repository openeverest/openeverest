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

package telemetry

import (
	"context"
	"fmt"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/internal/inventory"
)

// SummarySource provides the inventory summaries of some of the clusters an
// installation manages. The local cluster is one source; each multicluster
// agent transport (agents pushing summaries, summaries in cluster
// registrations, the main installation pulling them or reading the clusters
// directly) is another, and the report doesn't change with it.
type SummarySource interface {
	Summaries(ctx context.Context) ([]inventory.Summary, error)
}

// listPageSize bounds each list request telemetry sends to the API server.
const listPageSize = 500

// LocalSource summarizes the cluster the reader reads. With a direct API
// client it lists in pages, so no single request asks for a whole kind.
type LocalSource struct {
	Reader client.Reader
}

// Summaries implements SummarySource.
func (s LocalSource) Summaries(ctx context.Context) ([]inventory.Summary, error) {
	clusterID, err := ClusterID(ctx, s.Reader)
	if err != nil {
		return nil, err
	}
	snap, err := inventory.Collect(ctx, pagedReader{Reader: s.Reader, pageSize: listPageSize})
	if err != nil {
		return nil, err
	}
	return []inventory.Summary{inventory.Summarize(snap, clusterID, time.Now())}, nil
}

// pagedReader lists through the wrapped reader pageSize objects at a time.
type pagedReader struct {
	client.Reader

	pageSize int64
}

// List implements client.Reader.
func (r pagedReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	var items []runtime.Object
	continueToken := ""
	for {
		pageOpts := append(slices.Clone(opts), client.Limit(r.pageSize), client.Continue(continueToken))
		if err := r.Reader.List(ctx, list, pageOpts...); err != nil {
			return err
		}
		page, err := meta.ExtractList(list)
		if err != nil {
			return err
		}
		// The items point into the list, which the next page is decoded into.
		for _, item := range page {
			items = append(items, item.DeepCopyObject())
		}
		if continueToken = list.GetContinue(); continueToken == "" {
			break
		}
	}
	return meta.SetList(list, items)
}

// ClusterID returns the UID of the kube-system namespace, the identity of a
// Kubernetes cluster that monitoring labels use too.
func ClusterID(ctx context.Context, r client.Reader) (string, error) {
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: "kube-system"}, ns); err != nil {
		return "", fmt.Errorf("get the kube-system namespace: %w", err)
	}
	return string(ns.UID), nil
}
