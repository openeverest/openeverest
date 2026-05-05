package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

func TestValidateVersion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		version string
		engine  *everestv1alpha1.DatabaseEngine
		err     error
	}{
		{
			name:    "empty version is allowed",
			version: "",
			engine:  nil,
			err:     nil,
		},
		{
			name:    "shall exist in availableVersions",
			version: "8.0.32",
			engine: &everestv1alpha1.DatabaseEngine{
				Status: everestv1alpha1.DatabaseEngineStatus{
					AvailableVersions: everestv1alpha1.Versions{
						Engine: everestv1alpha1.ComponentsMap{
							"8.0.32": &everestv1alpha1.Component{},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:    "shall not exist in availableVersions",
			version: "8.0.32",
			engine: &everestv1alpha1.DatabaseEngine{
				Status: everestv1alpha1.DatabaseEngineStatus{
					AvailableVersions: everestv1alpha1.Versions{
						Engine: everestv1alpha1.ComponentsMap{
							"8.0.31": &everestv1alpha1.Component{},
						},
					},
				},
			},
			err: errors.New("8.0.32 is not in available versions list"),
		},
		{
			name:    "shall exist in allowedVersions",
			version: "8.0.32",
			engine: &everestv1alpha1.DatabaseEngine{
				Spec: everestv1alpha1.DatabaseEngineSpec{
					Type:            "pxc",
					AllowedVersions: []string{"8.0.32"},
				},
			},
			err: nil,
		},
		{
			name:    "shall not exist in allowedVersions",
			version: "8.0.32",
			engine: &everestv1alpha1.DatabaseEngine{
				Spec: everestv1alpha1.DatabaseEngineSpec{
					Type:            "pxc",
					AllowedVersions: []string{"8.0.31"},
				},
			},
			err: errors.New("using 8.0.32 version for pxc is not allowed"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateVersion(tc.version, tc.engine)
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			assert.Equal(t, err.Error(), tc.err.Error())
		})
	}
}

func TestValidateDBEngineUpgrade(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		engineType everestv1alpha1.EngineType
		oldVersion string
		newVersion string
		err        error
	}{
		{
			name:       "invalid version",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			oldVersion: "1.0.0",
			newVersion: "1!00;",
			err:        errInvalidVersion,
		},
		{
			name:       "major upgrade PXC",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			oldVersion: "8.0.22",
			newVersion: "9.0.0",
			err:        errDBEngineMajorVersionUpgrade,
		},
		{
			name:       "major upgrade PSMDB",
			engineType: everestv1alpha1.DatabaseEnginePSMDB,
			oldVersion: "8.0.22",
			newVersion: "9.0.0",
			err:        nil,
		},
		{
			name:       "skipping major upgrade PSMDB",
			engineType: everestv1alpha1.DatabaseEnginePSMDB,
			oldVersion: "8.0.22",
			newVersion: "10.0.0",
			err:        errDBEngineMajorUpgradeNotSeq,
		},
		{
			name:       "downgrade",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			oldVersion: "8.0.22",
			newVersion: "8.0.21",
			err:        errDBEngineDowngrade,
		},
		{
			name:       "valid upgrade",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			oldVersion: "8.0.22",
			newVersion: "8.0.23",
			err:        nil,
		},
		{
			name:       "valid upgrade (with 'v' prefix)",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			oldVersion: "v8.0.22",
			newVersion: "v8.0.23",
			err:        nil,
		},
		{
			name:       "major version downgrade",
			engineType: everestv1alpha1.DatabaseEnginePXC,
			oldVersion: "16.1",
			newVersion: "15.5",
			err:        errDBEngineDowngrade,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateDBEngineVersionUpgrade(tc.engineType, tc.newVersion, tc.oldVersion)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}
