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

package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func suspended(in *v1alpha1.Instance) {
	in.Spec.Suspend = &v1alpha1.SuspendSpec{Suspended: true}
}

func scaled(in *v1alpha1.Instance) {
	engine := in.Spec.Components["engine"]
	engine.Replicas = new(int32(5))
	in.Spec.Components["engine"] = engine
}

func TestRunSuspend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		unsupported bool
		old         *v1alpha1.Instance
		instance    *v1alpha1.Instance
		want        []string
	}{
		{
			name:     "suspend on create",
			instance: replicaSet(suspended),
		},
		{
			name:        "a provider without suspend support",
			unsupported: true,
			instance:    replicaSet(suspended),
			want: []string{
				"spec.suspend.suspended: Forbidden: provider psmdb does not support suspending instances",
			},
		},
		{
			name:        "suspended false needs no support",
			unsupported: true,
			instance: replicaSet(func(in *v1alpha1.Instance) {
				in.Spec.Suspend = &v1alpha1.SuspendSpec{}
			}),
		},
		{
			name:     "suspend alone",
			old:      replicaSet(nil),
			instance: replicaSet(suspended),
		},
		{
			name:     "resume alone",
			old:      replicaSet(suspended),
			instance: replicaSet(nil),
		},
		{
			name:     "metadata changes while suspended",
			old:      replicaSet(suspended),
			instance: replicaSet(func(in *v1alpha1.Instance) { suspended(in); in.Labels = map[string]string{"team": "a"} }),
		},
		{
			name:     "changes while running",
			old:      replicaSet(nil),
			instance: replicaSet(scaled),
		},
		{
			name: "changes while suspended",
			old:  replicaSet(suspended),
			instance: replicaSet(func(in *v1alpha1.Instance) {
				suspended(in)
				scaled(in)
				in.Spec.Maintenance = &v1alpha1.MaintenanceSpec{Approved: "upgrade-to-0.3"}
			}),
			want: []string{
				"spec.components: Forbidden: cannot change while the instance is suspended; resume it first",
				"spec.maintenance: Forbidden: cannot change while the instance is suspended; resume it first",
			},
		},
		{
			name:     "changes in the update that suspends",
			old:      replicaSet(nil),
			instance: replicaSet(func(in *v1alpha1.Instance) { suspended(in); scaled(in) }),
			want: []string{
				"spec.components: Forbidden: cannot change in the update that suspends the instance",
			},
		},
		{
			name: "changes in the update that resumes",
			old:  replicaSet(suspended),
			instance: replicaSet(func(in *v1alpha1.Instance) {
				in.Spec.Version = "7.0"
			}),
			want: []string{
				"spec.version: Forbidden: cannot change in the update that resumes the instance",
			},
		},
		{
			name: "deletion policy before deleting a suspended instance",
			old:  replicaSet(suspended),
			instance: replicaSet(func(in *v1alpha1.Instance) {
				suspended(in)
				in.Spec.DeletionPolicy = v1alpha1.InstanceDeletionPolicyOrphan
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			provider := testProvider(t)
			provider.Spec.Suspend = &v1alpha1.ProviderSuspend{}
			if tc.unsupported {
				provider.Spec.Suspend = nil
			}
			for _, in := range []*v1alpha1.Instance{tc.old, tc.instance} {
				if in != nil {
					in.Spec.ProviderRef.Name = "psmdb"
				}
			}
			v := &recordingValidator{}

			res := Run(t.Context(), v, controller.ValidationInput{Instance: tc.instance, Old: tc.old, Provider: provider})

			assert.Equal(t, tc.want, errorStrings(res.Errors))
			assert.Equal(t, len(tc.want) == 0, v.called, "Validate runs only when the declared checks pass")
		})
	}
}
