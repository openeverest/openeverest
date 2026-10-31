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

package v1alpha1

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
)

// ErrInvalidEndpoint is returned by ParseEndpoint for anything but an
// absolute http or https URL with a host and without user information.
var ErrInvalidEndpoint = errors.New("invalid endpoint")

// Endpoint is an http or https service URL split into the parts executors
// render. The scheme is the plain-HTTP switch: there is no separate flag.
//
// +kubebuilder:object:generate=false
type Endpoint struct {
	// Host is the hostname or IP address, without brackets.
	Host string
	// Port is the explicit port, or the scheme's default (443 or 80).
	Port int
	// Secure is true for https.
	Secure bool
}

// HostPort returns host:port, bracketing IPv6 addresses.
func (e Endpoint) HostPort() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

// ParseEndpoint parses an absolute http or https URL.
func ParseEndpoint(raw string) (Endpoint, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Endpoint{}, fmt.Errorf("%w: %w", ErrInvalidEndpoint, err)
	}
	var ep Endpoint
	switch u.Scheme {
	case "https":
		ep.Secure, ep.Port = true, 443
	case "http":
		ep.Port = 80
	default:
		return Endpoint{}, fmt.Errorf("%w: %q is not an http or https URL", ErrInvalidEndpoint, raw)
	}
	if u.User != nil {
		return Endpoint{}, fmt.Errorf("%w: %q carries user information", ErrInvalidEndpoint, raw)
	}
	ep.Host = u.Hostname()
	if ep.Host == "" {
		return Endpoint{}, fmt.Errorf("%w: %q has no host", ErrInvalidEndpoint, raw)
	}
	if p := u.Port(); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil || port < 1 || port > 65535 {
			return Endpoint{}, fmt.Errorf("%w: %q has an invalid port", ErrInvalidEndpoint, raw)
		}
		ep.Port = port
	}
	return ep, nil
}
