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

package monitoring

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// instanceMonitoringDestinationField indexes Instances by the name of every
// MonitoringDestination their spec.monitoring.destinations lists.
const instanceMonitoringDestinationField = ".spec.monitoring.destinations.destinationRef.name"

const destinationKind = "MonitoringDestination"

// SetupIndexes registers the field indexes several monitoring controllers
// list by. It runs once, before the controllers are set up: an index cannot
// be registered twice.
func SetupIndexes(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1alpha1.Instance{},
		instanceMonitoringDestinationField, instanceMonitoringDestinationNames,
	); err != nil {
		return fmt.Errorf("indexing instance by monitoring destination: %w", err)
	}
	return nil
}

// instanceMonitoringDestinationNames is the extractor for
// instanceMonitoringDestinationField.
func instanceMonitoringDestinationNames(obj client.Object) []string {
	in, ok := obj.(*corev1alpha1.Instance)
	if !ok {
		return nil
	}
	return destinationNames(in)
}

// destinationNames returns the names of the MonitoringDestinations the
// Instance lists. Entries of another kind are left out, so they can never
// be mistaken for a namesake MonitoringDestination.
func destinationNames(in *corev1alpha1.Instance) []string {
	entries := instanceMonitoringDestinations(in)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if destinationRefOf(e).Kind == destinationKind {
			names = append(names, e.DestinationRef.Name)
		}
	}
	return names
}

// instanceMonitoringDestinations returns the Instance's destination entries.
func instanceMonitoringDestinations(in *corev1alpha1.Instance) []corev1alpha1.InstanceMonitoringDestination {
	if in.Spec.Monitoring == nil {
		return nil
	}
	return in.Spec.Monitoring.Destinations
}
