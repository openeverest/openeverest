// everest
// Copyright (C) 2025 Percona LLC
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
	"os"
	"time"

	"github.com/scarf-sh/scarf-go/scarf"
)

// Transport delivers a single telemetry event consisting of string properties.
// Implementations must never block or fail the caller for long; honor ctx.
type Transport interface {
	Send(ctx context.Context, props map[string]string) error
}

// defaultScarfTimeout caps a single Scarf request. Scarf SDK default is 3s;
// we keep it explicit to make the budget obvious.
const defaultScarfTimeout = 3 * time.Second

// scarfTransport sends events to a Scarf Gateway Event Collection endpoint.
// This is the only place that imports the Scarf SDK.
type scarfTransport struct {
	logger *scarf.ScarfEventLogger
}

// NewScarfTransport builds a Transport backed by the Scarf SDK.
//
// SCARF_NO_ANALYTICS is suppressed so the public opt-out surface stays
// vendor-agnostic: users disable telemetry via --disable-telemetry,
// DISABLE_TELEMETRY=true, or the standard DO_NOT_TRACK.
func NewScarfTransport(endpointURL string) Transport {
	_ = os.Unsetenv("SCARF_NO_ANALYTICS")
	return &scarfTransport{
		logger: scarf.NewScarfEventLogger(endpointURL, defaultScarfTimeout),
	}
}

func (s *scarfTransport) Send(ctx context.Context, props map[string]string) error {
	payload := make(map[string]any, len(props))
	for k, v := range props {
		payload[k] = v
	}
	// LogEventWithTimeout uses its own timeout; ctx is honored by the caller
	// via the surrounding goroutine lifecycle.
	_ = ctx
	return s.logger.LogEventWithTimeout(payload, defaultScarfTimeout)
}
