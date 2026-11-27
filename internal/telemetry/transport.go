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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	sendTimeout = 5 * time.Second
	// maxErrorBody bounds how much of a rejected request's body is logged.
	maxErrorBody = 1024
)

// Transport delivers a report.
type Transport interface {
	Send(ctx context.Context, r Report) error
}

// HTTPTransport POSTs the report as a JSON object. Scarf event collection
// accepts it as is, and so does any collector that takes a JSON POST.
type HTTPTransport struct {
	URL       string
	UserAgent string
	Client    *http.Client
}

// NewHTTPTransport returns an HTTPTransport with a bounded timeout.
func NewHTTPTransport(url, userAgent string) *HTTPTransport {
	return &HTTPTransport{URL: url, UserAgent: userAgent, Client: &http.Client{Timeout: sendTimeout}}
}

// Send implements Transport.
func (t *HTTPTransport) Send(ctx context.Context, r Report) error {
	body, err := json.Marshal(r)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", t.UserAgent)
	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return fmt.Errorf("telemetry endpoint answered %s: %s", resp.Status, bytes.TrimSpace(msg))
	}
	return nil
}

// LogTransport writes the report to the log instead of sending it, so a user
// can see exactly what would leave the cluster.
type LogTransport struct {
	Log *zap.SugaredLogger
}

// Send implements Transport.
func (t LogTransport) Send(_ context.Context, r Report) error {
	body, err := json.Marshal(r)
	if err != nil {
		return err
	}
	t.Log.Infow("Telemetry dry run: this report would be sent", "report", string(body))
	return nil
}
