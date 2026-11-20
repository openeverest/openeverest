// everest
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

package kubernetes

import (
	"context"
	"slices"
	"sync"

	"k8s.io/client-go/rest"
)

type warningCollectorKey struct{}

// WarningCollector gathers the warnings the Kubernetes API returns to the calls made with its context.
type WarningCollector struct {
	mu       sync.Mutex
	warnings []string
}

// WithWarningCollector returns a copy of ctx whose Kubernetes API warnings are gathered in the returned collector.
func WithWarningCollector(ctx context.Context) (context.Context, *WarningCollector) {
	c := &WarningCollector{}
	return context.WithValue(ctx, warningCollectorKey{}, c), c
}

// Warnings returns the distinct warning texts in the order they arrived.
func (c *WarningCollector) Warnings() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.warnings)
}

func (c *WarningCollector) add(text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !slices.Contains(c.warnings, text) {
		c.warnings = append(c.warnings, text)
	}
}

// CollectingWarningHandler adds each warning to the WarningCollector of the call's context, if any, and passes it on to Next.
type CollectingWarningHandler struct {
	Next rest.WarningHandlerWithContext
}

// HandleWarningHeaderWithContext implements rest.WarningHandlerWithContext.
func (h CollectingWarningHandler) HandleWarningHeaderWithContext(ctx context.Context, code int, agent, text string) {
	if code == 299 {
		AddWarning(ctx, text)
	}
	h.Next.HandleWarningHeaderWithContext(ctx, code, agent, text)
}

// AddWarning adds text to the WarningCollector of ctx, if any, as if the Kubernetes API had returned it, for a warning
// the caller raises itself. A text the Kubernetes API returns too is reported once.
func AddWarning(ctx context.Context, text string) {
	if c, ok := ctx.Value(warningCollectorKey{}).(*WarningCollector); ok && text != "" {
		c.add(text)
	}
}
