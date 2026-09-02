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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// targetSpec returns a target ProviderSpec modeled on the PSMDB example from
// spec 009 §9: provider release 0.3 where mongod 6.0.19-16 is removed and
// 7.0.18-11 is deprecated with a runway.
func targetSpec() *v1alpha1.ProviderSpec {
	return &v1alpha1.ProviderSpec{
		Release: &v1alpha1.Release{
			Version:           "0.3",
			MinUpgradableFrom: "0.2",
		},
		Components: map[string]v1alpha1.Component{
			"engine": {Type: "mongod"},
			"proxy":  {Type: "mongos"},
		},
		ComponentTypes: map[string]v1alpha1.ComponentType{
			"mongod": {Versions: []v1alpha1.ComponentVersion{
				{Version: "6.0.19-16", Deprecated: true, RemovedInVersion: "0.3"},
				{Version: "7.0.18-11", Deprecated: true, RemovedInVersion: "0.5"},
				{Version: "8.0.12-4", Default: true},
			}},
			"mongos": {Versions: []v1alpha1.ComponentVersion{
				{Version: "8.0.12-4", Default: true},
			}},
		},
		Versions: []v1alpha1.VersionBundle{
			{Name: "6.0.19", Components: map[string]string{"engine": "6.0.19-16"}},
			{Name: "7.0.18", Components: map[string]string{"engine": "7.0.18-11"}},
			{Name: "8.0.12", Default: true, Components: map[string]string{"engine": "8.0.12-4", "proxy": "8.0.12-4"}},
		},
	}
}

func instance(name string, mutate func(*v1alpha1.Instance)) v1alpha1.Instance {
	in := v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "db"},
		Spec: v1alpha1.InstanceSpec{
			Components: map[string]v1alpha1.ComponentSpec{
				"engine": {},
			},
		},
	}
	if mutate != nil {
		mutate(&in)
	}
	return in
}

func TestCheckUpgradePath(t *testing.T) {
	t.Parallel()

	release := func(version string) *v1alpha1.ProviderSpec {
		return &v1alpha1.ProviderSpec{Release: &v1alpha1.Release{Version: version}}
	}

	tests := []struct {
		name         string
		current      *v1alpha1.ProviderSpec
		target       *v1alpha1.ProviderSpec
		wantSeverity UpgradeSeverity
		wantReason   string
	}{
		{
			name:    "no floor declared passes",
			current: release("0.1"),
			target:  release("0.3"),
		},
		{
			name:    "floor satisfied passes",
			current: release("0.2"),
			target:  targetSpec(),
		},
		{
			name:    "installed above floor passes",
			current: release("0.2.5"),
			target:  targetSpec(),
		},
		{
			name:         "installed below floor blocks",
			current:      release("0.1"),
			target:       targetSpec(),
			wantSeverity: UpgradeError,
			wantReason:   UpgradeReasonPathBlocked,
		},
		{
			name:         "no installed release version warns",
			current:      &v1alpha1.ProviderSpec{},
			target:       targetSpec(),
			wantSeverity: UpgradeWarning,
			wantReason:   UpgradeReasonPathUnverified,
		},
		{
			name:         "nil current spec warns",
			current:      nil,
			target:       targetSpec(),
			wantSeverity: UpgradeWarning,
			wantReason:   UpgradeReasonPathUnverified,
		},
		{
			name:         "unparseable installed version warns",
			current:      release("not-a-version"),
			target:       targetSpec(),
			wantSeverity: UpgradeWarning,
			wantReason:   UpgradeReasonPathUnverified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			issues := CheckUpgradePath(tt.current, tt.target)

			if tt.wantSeverity == "" {
				assert.Empty(t, issues)
				return
			}
			require.Len(t, issues, 1)
			assert.Equal(t, tt.wantSeverity, issues[0].Severity)
			assert.Equal(t, tt.wantReason, issues[0].Reason)
		})
	}
}

func TestPreflightUpgrade_ComponentVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		in           v1alpha1.Instance
		wantSeverity UpgradeSeverity
		wantReason   string
	}{
		{
			name: "supported explicit version passes",
			in: instance("mongo-prod", func(in *v1alpha1.Instance) {
				in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Version: "8.0.12-4"}
			}),
		},
		{
			name: "version absent from catalog blocks",
			in: instance("mongo-old", func(in *v1alpha1.Instance) {
				in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Version: "5.0.1-1"}
			}),
			wantSeverity: UpgradeError,
			wantReason:   UpgradeReasonVersionRemoved,
		},
		{
			name: "version removed in target release blocks even when still listed",
			in: instance("mongo-legacy", func(in *v1alpha1.Instance) {
				in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Version: "6.0.19-16"}
			}),
			wantSeverity: UpgradeError,
			wantReason:   UpgradeReasonVersionRemoved,
		},
		{
			name: "deprecated with future removal warns",
			in: instance("mongo-mid", func(in *v1alpha1.Instance) {
				in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Version: "7.0.18-11"}
			}),
			wantSeverity: UpgradeWarning,
			wantReason:   UpgradeReasonVersionDeprecated,
		},
		{
			name: "component unknown to target blocks",
			in: instance("mongo-extra", func(in *v1alpha1.Instance) {
				in.Spec.Components["analytics"] = v1alpha1.ComponentSpec{Version: "1.0.0"}
				delete(in.Spec.Components, "engine")
			}),
			wantSeverity: UpgradeError,
			wantReason:   UpgradeReasonComponentUnsupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			issues := PreflightUpgrade(targetSpec(), []v1alpha1.Instance{tt.in})

			if tt.wantSeverity == "" {
				assert.Empty(t, issues)
				return
			}
			require.Len(t, issues, 1)
			assert.Equal(t, tt.wantSeverity, issues[0].Severity)
			assert.Equal(t, tt.wantReason, issues[0].Reason)
			assert.Equal(t, tt.in.Name, issues[0].InstanceName)
			assert.Equal(t, "db", issues[0].Namespace)
		})
	}
}

func TestPreflightUpgrade_BundleResolution(t *testing.T) {
	t.Parallel()

	t.Run("frozen status bundle resolves component versions", func(t *testing.T) {
		t.Parallel()

		// status.version frozen to a bundle whose engine version is removed
		// in the target: must block, mirroring the reconciler's resolution.
		in := instance("mongo-frozen", func(in *v1alpha1.Instance) {
			in.Status.Version = "6.0.19"
		})

		issues := PreflightUpgrade(targetSpec(), []v1alpha1.Instance{in})

		require.Len(t, issues, 1)
		assert.Equal(t, UpgradeError, issues[0].Severity)
		assert.Equal(t, UpgradeReasonVersionRemoved, issues[0].Reason)
		assert.Equal(t, "engine", issues[0].Component)
	})

	t.Run("explicit spec version takes precedence over frozen bundle", func(t *testing.T) {
		t.Parallel()

		in := instance("mongo-explicit", func(in *v1alpha1.Instance) {
			in.Spec.Version = "8.0.12"
			in.Status.Version = "6.0.19"
		})

		issues := PreflightUpgrade(targetSpec(), []v1alpha1.Instance{in})

		assert.Empty(t, issues)
	})

	t.Run("bundle missing from target catalog blocks", func(t *testing.T) {
		t.Parallel()

		in := instance("mongo-gone", func(in *v1alpha1.Instance) {
			in.Status.Version = "4.4.0"
		})

		issues := PreflightUpgrade(targetSpec(), []v1alpha1.Instance{in})

		require.Len(t, issues, 1)
		assert.Equal(t, UpgradeError, issues[0].Severity)
		assert.Equal(t, UpgradeReasonVersionRemoved, issues[0].Reason)
		assert.Contains(t, issues[0].Message, "4.4.0")
	})

	t.Run("no version anywhere falls back to target default and passes", func(t *testing.T) {
		t.Parallel()

		in := instance("mongo-new", nil)

		issues := PreflightUpgrade(targetSpec(), []v1alpha1.Instance{in})

		assert.Empty(t, issues)
	})

	t.Run("deleting instance is skipped", func(t *testing.T) {
		t.Parallel()

		now := metav1.Now()
		in := instance("mongo-deleting", func(in *v1alpha1.Instance) {
			in.DeletionTimestamp = &now
			in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Version: "6.0.19-16"}
		})

		issues := PreflightUpgrade(targetSpec(), []v1alpha1.Instance{in})

		assert.Empty(t, issues)
	})
}

func TestPreflightUpgrade_MultipleInstancesAndComponents(t *testing.T) {
	t.Parallel()

	legacy := instance("mongo-legacy", func(in *v1alpha1.Instance) {
		in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Version: "6.0.19-16"}
		in.Spec.Components["proxy"] = v1alpha1.ComponentSpec{Version: "8.0.12-4"}
	})
	prod := instance("mongo-prod", func(in *v1alpha1.Instance) {
		in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Version: "8.0.12-4"}
	})

	issues := PreflightUpgrade(targetSpec(), []v1alpha1.Instance{legacy, prod})

	require.Len(t, issues, 1)
	assert.Equal(t, "mongo-legacy", issues[0].InstanceName)
	assert.Equal(t, "engine", issues[0].Component)
	assert.True(t, HasBlockingIssues(issues))
}

func TestHasBlockingIssues(t *testing.T) {
	t.Parallel()

	assert.False(t, HasBlockingIssues(nil))
	assert.False(t, HasBlockingIssues([]UpgradeIssue{{Severity: UpgradeWarning}}))
	assert.True(t, HasBlockingIssues([]UpgradeIssue{{Severity: UpgradeWarning}, {Severity: UpgradeError}}))
}

// upgradeCheckingProvider implements UpgradeProvider for testing
// RunUpgradePreflight's dispatch.
type upgradeCheckingProvider struct {
	BaseProvider

	issues []UpgradeIssue
}

func (p *upgradeCheckingProvider) Validate(*Context) error         { return nil }
func (p *upgradeCheckingProvider) Sync(*Context) error             { return nil }
func (p *upgradeCheckingProvider) Status(*Context) (Status, error) { return Status{}, nil }
func (p *upgradeCheckingProvider) Cleanup(*Context) error          { return nil }
func (p *upgradeCheckingProvider) CheckUpgrade(*Context, *v1alpha1.ProviderSpec, []v1alpha1.Instance) []UpgradeIssue {
	return p.issues
}

// plainProvider does not implement UpgradeProvider.
type plainProvider struct {
	BaseProvider
}

func (p *plainProvider) Validate(*Context) error         { return nil }
func (p *plainProvider) Sync(*Context) error             { return nil }
func (p *plainProvider) Status(*Context) (Status, error) { return Status{}, nil }
func (p *plainProvider) Cleanup(*Context) error          { return nil }

func TestRunUpgradePreflight(t *testing.T) {
	t.Parallel()

	target := targetSpec()
	current := &v1alpha1.ProviderSpec{Release: &v1alpha1.Release{Version: "0.1"}}
	instances := []v1alpha1.Instance{
		instance("mongo-legacy", func(in *v1alpha1.Instance) {
			in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Version: "6.0.19-16"}
		}),
	}

	t.Run("aggregates path, catalog, and provider issues", func(t *testing.T) {
		t.Parallel()

		provider := &upgradeCheckingProvider{issues: []UpgradeIssue{{
			Severity: UpgradeError,
			Reason:   "SkewWindowExceeded",
		}}}

		issues := RunUpgradePreflight(nil, provider, current, target, instances)

		require.Len(t, issues, 3)
		assert.Equal(t, UpgradeReasonPathBlocked, issues[0].Reason)
		assert.Equal(t, UpgradeReasonVersionRemoved, issues[1].Reason)
		assert.Equal(t, "SkewWindowExceeded", issues[2].Reason)
	})

	t.Run("provider without UpgradeProvider runs generic checks only", func(t *testing.T) {
		t.Parallel()

		issues := RunUpgradePreflight(nil, &plainProvider{}, current, target, instances)

		require.Len(t, issues, 2)
		assert.Equal(t, UpgradeReasonPathBlocked, issues[0].Reason)
		assert.Equal(t, UpgradeReasonVersionRemoved, issues[1].Reason)
	})
}
