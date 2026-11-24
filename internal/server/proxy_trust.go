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

package server

import (
	"fmt"
	"net"
	"strings"

	"github.com/labstack/echo/v4"
)

// configureProxyTrust makes the client IP and the request scheme come from X-Forwarded-For
// and X-Forwarded-Proto only when a trusted proxy sent them. Rate limits are keyed by client
// IP and the refresh-token cookie is marked Secure by scheme, so without this every client
// behind one ingress would share its address, and any client could spoof the headers.
//
// Loopback, link-local and private peers are trusted by default, which covers an in-cluster
// ingress controller. A proxy that connects from a public address is trusted only when its
// CIDR is listed in trustedProxies.
func configureProxyTrust(e *echo.Echo, trustedProxies []string) error {
	options := make([]echo.TrustOption, 0, len(trustedProxies))
	for _, cidr := range trustedProxies {
		_, ipRange, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil {
			return fmt.Errorf("invalid TRUSTED_PROXIES entry %q: %w", cidr, err)
		}
		options = append(options, echo.TrustIPRange(ipRange))
	}

	e.IPExtractor = echo.ExtractIPFromXFFHeader(options...)
	e.SchemeExtractor = echo.ExtractSchemeFromHeaders(options...)
	return nil
}
