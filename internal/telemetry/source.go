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
	"time"

	corev1 "k8s.io/api/core/v1"
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

// LocalSource summarizes the cluster the reader reads.
type LocalSource struct {
	Reader client.Reader
}

// Summaries implements SummarySource.
func (s LocalSource) Summaries(ctx context.Context) ([]inventory.Summary, error) {
	clusterID, err := ClusterID(ctx, s.Reader)
	if err != nil {
		return nil, err
	}
	snap, err := inventory.Collect(ctx, s.Reader)
	if err != nil {
		return nil, err
	}
	return []inventory.Summary{inventory.Summarize(snap, clusterID, time.Now())}, nil
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
