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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/inventory"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	const url = "https://telemetry.example.com"
	for _, tc := range []struct {
		name     string
		config   Config
		want     Settings
		disabled bool
		invalid  bool
	}{
		{name: "defaults", config: Config{URL: url}, want: Settings{URL: url, Interval: DefaultInterval}},
		{name: "interval", config: Config{URL: url, Interval: "6h"}, want: Settings{URL: url, Interval: 6 * time.Hour}},
		{name: "dry run needs no endpoint", config: Config{DryRun: true}, want: Settings{Interval: DefaultInterval, DryRun: true}},
		{name: "DISABLE_TELEMETRY", config: Config{URL: url, Disabled: true}, disabled: true},
		{name: "DISABLE_TELEMETRY wins over dry run", config: Config{DryRun: true, Disabled: true}, disabled: true},
		{name: "DO_NOT_TRACK=1", config: Config{URL: url, DoNotTrack: "1"}, disabled: true},
		{name: "DO_NOT_TRACK=yes", config: Config{URL: url, DoNotTrack: "yes"}, disabled: true},
		{name: "DO_NOT_TRACK=0", config: Config{URL: url, DoNotTrack: "0"}, want: Settings{URL: url, Interval: DefaultInterval}},
		{name: "DO_NOT_TRACK=false", config: Config{URL: url, DoNotTrack: "false"}, want: Settings{URL: url, Interval: DefaultInterval}},
		{name: "a build without an endpoint", config: Config{}, disabled: true},
		{name: "interval under an hour", config: Config{URL: url, Interval: "0s"}, invalid: true},
		{name: "unparsable interval", config: Config{URL: url, Interval: "daily"}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Resolve(tc.config)
			switch {
			case tc.disabled:
				require.ErrorIs(t, err, ErrDisabled)
			case tc.invalid:
				require.Error(t, err)
				require.NotErrorIs(t, err, ErrDisabled)
			default:
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestHTTPTransport(t *testing.T) {
	t.Parallel()

	var got Report
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	require.NoError(t, NewHTTPTransport(srv.URL, "test").Send(context.Background(), Report{"event": "heartbeat"}))
	assert.Equal(t, Report{"event": "heartbeat"}, got)
}

func TestHTTPTransportReportsRejection(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "unknown package")
	}))
	t.Cleanup(srv.Close)

	err := NewHTTPTransport(srv.URL, "test").Send(context.Background(), Report{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "unknown package")
}

type staticSource struct {
	summaries []inventory.Summary
	err       error
}

func (s staticSource) Summaries(context.Context) ([]inventory.Summary, error) {
	return s.summaries, s.err
}

type recordingTransport struct{ reports []Report }

func (t *recordingTransport) Send(_ context.Context, r Report) error {
	t.reports = append(t.reports, r)
	return nil
}

func installation(context.Context) (Installation, error) {
	return Installation{ID: "install-1", BuildChannel: "release", Version: "2.0.0", Contract: "2.0.0"}, nil
}

func TestRunnerAddsUpEverySource(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	transport := &recordingTransport{}
	r := &Runner{
		Sources: []SummarySource{
			staticSource{summaries: []inventory.Summary{{
				ClusterID: "main", CollectedAt: now, Instances: 2,
				Providers: []string{"valkey"}, ProviderInstallations: 1,
				InstancesByProvider: map[string]int{"valkey": 2},
			}}},
			staticSource{err: errors.New("agent unreachable")},
			staticSource{summaries: []inventory.Summary{
				{ClusterID: "agent-1", CollectedAt: now, Instances: 3, InstancesByProvider: map[string]int{inventory.CustomProvider: 3}},
				{ClusterID: "agent-2", CollectedAt: now.Add(-100 * time.Hour), Instances: 9},
			}},
		},
		Installation: installation,
		Transport:    transport,
		Log:          zap.NewNop().Sugar(),
		Now:          func() time.Time { return now },
	}

	require.NoError(t, r.SendOnce(context.Background()))
	require.Len(t, transport.reports, 1)
	report := transport.reports[0]
	assert.Equal(t, "heartbeat", report["event"])
	assert.Equal(t, "install-1", report["instance_id"])
	assert.Equal(t, "release", report["build_channel"])
	assert.Equal(t, "2", report["clusters_total"])
	assert.Equal(t, "1", report["clusters_stale"])
	assert.Equal(t, "5", report["instances_total"])
	assert.Equal(t, "1", report["providers_distinct"])
	assert.Equal(t, "2", report["provider.valkey.instances"])
	assert.Equal(t, "3", report["provider.custom.instances"])
}

func TestRunnerSendsNothingWithoutSummaries(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	r := &Runner{
		Sources:      []SummarySource{staticSource{err: errors.New("down")}},
		Installation: installation,
		Transport:    transport,
		Log:          zap.NewNop().Sugar(),
		Now:          time.Now,
	}

	require.Error(t, r.SendOnce(context.Background()))
	assert.Empty(t, transport.reports)
}

// TestReportCarriesNoUserChosenNames builds a report from a cluster whose
// every name is user-chosen, except one official provider, and checks that
// none of those names leaves the cluster.
func TestReportCarriesNoUserChosenNames(t *testing.T) {
	t.Parallel()

	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(corev1alpha1.AddToScheme(s))
	utilruntime.Must(extensionsv1alpha1.AddToScheme(s))
	utilruntime.Must(backupv1alpha1.AddToScheme(s))
	utilruntime.Must(monitoringv1alpha1.AddToScheme(s))

	const secret = "acme-secret"
	objects := []client.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: types.UID("cluster-uid")}},
		&corev1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: secret + "-provider"}},
		&corev1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "valkey"}},
		&extensionsv1alpha1.Plugin{ObjectMeta: metav1.ObjectMeta{Name: secret + "-plugin"}},
		&extensionsv1alpha1.InstalledExtension{
			ObjectMeta: metav1.ObjectMeta{Name: secret + "-ext"},
			Spec:       extensionsv1alpha1.InstalledExtensionSpec{Type: extensionsv1alpha1.InstalledExtensionTypeProvider, CatalogID: secret + "-catalog"},
		},
		&corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: secret + "-db", Namespace: secret + "-ns"},
			Spec:       corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: secret + "-provider"}},
		},
		&corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: secret + "-cache", Namespace: secret + "-ns"},
			Spec:       corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: "valkey"}},
		},
		&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: secret + "-bucket", Namespace: secret + "-ns"}},
		&monitoringv1alpha1.MonitoringDestination{ObjectMeta: metav1.ObjectMeta{Name: secret + "-grafana", Namespace: secret + "-ns"}},
	}
	reader := fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build()
	transport := &recordingTransport{}
	r := &Runner{
		Sources:      []SummarySource{LocalSource{Reader: reader}},
		Installation: installation,
		Transport:    transport,
		Log:          zap.NewNop().Sugar(),
		Now:          time.Now,
	}

	require.NoError(t, r.SendOnce(context.Background()))
	require.Len(t, transport.reports, 1)
	body, err := json.Marshal(transport.reports[0])
	require.NoError(t, err)
	assert.NotContains(t, string(body), secret)
	assert.NotContains(t, string(body), "cluster-uid", "the cluster ID only deduplicates summaries")
	assert.Equal(t, "1", transport.reports[0]["provider.valkey.instances"])
	assert.Equal(t, "1", transport.reports[0]["provider.custom.instances"])
	assert.Equal(t, "2", transport.reports[0]["provider_installations"])
	assert.True(t, strings.HasPrefix(transport.reports[0]["providers"], "valkey"))
}

// pagingReader serves Instances page by page and decodes each page into the
// caller's list like the API client does, reusing its Items array.
type pagingReader struct {
	client.Reader

	pages [][]string
	limit int64
}

func (r *pagingReader) List(_ context.Context, list client.ObjectList, opts ...client.ListOption) error {
	o := (&client.ListOptions{}).ApplyOptions(opts)
	r.limit = o.Limit
	page := 0
	if o.Continue != "" {
		page = int(o.Continue[0] - '0')
	}
	l := list.(*corev1alpha1.InstanceList) //nolint:forcetypeassert
	l.Items = l.Items[:0]
	for _, name := range r.pages[page] {
		l.Items = append(l.Items, corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}
	l.Continue = ""
	if page+1 < len(r.pages) {
		l.Continue = string(rune('0' + page + 1))
	}
	return nil
}

func TestPagedReaderKeepsEveryPage(t *testing.T) {
	t.Parallel()

	src := &pagingReader{pages: [][]string{{"a", "b"}, {"c", "d"}, {"e"}}}
	var list corev1alpha1.InstanceList

	require.NoError(t, pagedReader{Reader: src, pageSize: 2}.List(context.Background(), &list))

	names := make([]string, 0, len(list.Items))
	for _, i := range list.Items {
		names = append(names, i.Name)
	}
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, names)
	assert.Equal(t, int64(2), src.limit)
	assert.Empty(t, list.Continue)
}

type blockingSource struct{}

func (blockingSource) Summaries(ctx context.Context) ([]inventory.Summary, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type panickingSource struct{}

func (panickingSource) Summaries(context.Context) ([]inventory.Summary, error) {
	panic("boom")
}

// TestRunnerNeverBlocksOrCrashes checks that a hung source is given up after
// the report timeout and that a panicking one is contained, so the loop
// keeps running.
func TestRunnerNeverBlocksOrCrashes(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	for _, src := range []SummarySource{blockingSource{}, panickingSource{}} {
		r := &Runner{
			Timeout:      50 * time.Millisecond,
			Sources:      []SummarySource{src},
			Installation: installation,
			Transport:    transport,
			Log:          zap.NewNop().Sugar(),
			Now:          time.Now,
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			r.report(context.Background())
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("a report with %T did not return", src)
		}
	}
	assert.Empty(t, transport.reports)
}

func TestHTTPTransportGivesUp(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})
	tr := NewHTTPTransport(srv.URL, "test")
	tr.Client.Timeout = 50 * time.Millisecond

	start := time.Now()
	require.Error(t, tr.Send(context.Background(), Report{}))
	assert.Less(t, time.Since(start), 5*time.Second)
}
