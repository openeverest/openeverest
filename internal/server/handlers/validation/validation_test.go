package validation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

func TestValidateMetadata(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		db      *everestv1alpha1.DatabaseCluster
		wantErr error
	}{
		{
			db:      &everestv1alpha1.DatabaseCluster{},
			wantErr: errEmptyNamespace,
		},

		{
			db: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			wantErr: errEmptyName,
		},
		{
			db: &everestv1alpha1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test",
				},
			},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		err := validateMetadata(tc.db)
		require.Equal(t, tc.wantErr, err)
	}
}

func TestContainsVersion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		version  string
		versions []string
		result   bool
	}{
		{
			version:  "1",
			versions: []string{},
			result:   false,
		},
		{
			version:  "1",
			versions: []string{"1", "2"},
			result:   true,
		},
		{
			version:  "1",
			versions: []string{"1"},
			result:   true,
		},
		{
			version:  "1",
			versions: []string{"12", "23"},
			result:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()
			res := containsVersion(tc.version, tc.versions)
			assert.Equal(t, res, tc.result)
		})
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(dateFormat, s)
	require.NoError(t, err)
	return tm
}
