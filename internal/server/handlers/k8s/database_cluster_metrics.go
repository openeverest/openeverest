package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/percona/everest/api"
)

// GetDatabaseClusterMetrics proxies Prometheus queries to the configured MonitoringConfig.
func (h *k8sHandler) GetDatabaseClusterMetrics(ctx context.Context, namespace, name string, params api.GetDatabaseClusterMetricsParams) (map[string]interface{}, error) {
	databaseCluster, err := h.kubeConnector.GetDatabaseCluster(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to get database cluster %s/%s: %w", namespace, name, err)
	}

	if databaseCluster.Spec.Monitoring == nil || databaseCluster.Spec.Monitoring.MonitoringConfigName == "" {
		return nil, fmt.Errorf("monitoring is not configured for database cluster %s", name)
	}

	monitoringConfig, err := h.kubeConnector.GetMonitoringConfig(ctx, types.NamespacedName{Namespace: namespace, Name: databaseCluster.Spec.Monitoring.MonitoringConfigName})
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring config: %w", err)
	}

	secretName := monitoringConfig.Spec.CredentialsSecretName
	secret, err := h.kubeConnector.GetSecret(ctx, types.NamespacedName{Namespace: namespace, Name: secretName})
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring credentials secret %s: %w", secretName, err)
	}

	apiKeyBytes, ok := secret.Data["apiKey"]
	if !ok {
		return nil, fmt.Errorf("apiKey not found in monitoring credentials secret %s", secretName)
	}

	// Build Prometheus query URL.
	// For PMM, the prometheus API is at /prometheus/api/v1/query_range
	baseURL := strings.TrimRight(monitoringConfig.Spec.PMM.URL, "/")
	promURL := fmt.Sprintf("%s/prometheus/api/v1/query_range", baseURL)

	reqURL, err := url.Parse(promURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse monitoring URL: %w", err)
	}

	q := reqURL.Query()
	q.Set("query", params.Query)
	q.Set("start", params.Start)
	q.Set("end", params.End)
	q.Set("step", params.Step)
	reqURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// Basic Auth using API Key
	httpReq.SetBasicAuth("api_key", string(apiKeyBytes))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute monitoring proxy request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("monitoring endpoint returned status %d: %s", resp.StatusCode, string(b))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode monitoring response: %w", err)
	}

	return result, nil
}
