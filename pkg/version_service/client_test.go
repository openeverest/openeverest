package versionservice_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/everest/pkg/version_service"
)

func TestGetSupportedEngineVersions_NilMatrix(t *testing.T) {
	t.Parallel()

	// Server returns a VersionResponse where versions slice has an entry with a nil matrix
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"versions": [{}]}`))
	}))
	defer ts.Close()

	client := versionservice.New(ts.URL)
	ctx := context.Background()

	versions, err := client.GetSupportedEngineVersions(ctx, versionservice.PXCOperatorName, "1.0.0")
	assert.Error(t, err)
	assert.Nil(t, versions)
	assert.Contains(t, err.Error(), "no version matrix found")
}

func TestGetSupportedEngineVersions_UnsupportedOperator(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"versions": [{"matrix": {"pxc": {"8.0.0": {}}}}]} `))
	}))
	defer ts.Close()

	client := versionservice.New(ts.URL)
	ctx := context.Background()

	versions, err := client.GetSupportedEngineVersions(ctx, "invalid-operator", "1.0.0")
	assert.Error(t, err)
	assert.Nil(t, versions)
	assert.Contains(t, err.Error(), "unsupported operator")
}

func TestGetSupportedEngineVersions_Success(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"versions": [{"matrix": {"pxc": {"8.0.32": {}}}}]} `))
	}))
	defer ts.Close()

	client := versionservice.New(ts.URL)
	ctx := context.Background()

	versions, err := client.GetSupportedEngineVersions(ctx, versionservice.PXCOperatorName, "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, []string{"8.0.32"}, versions)
}
