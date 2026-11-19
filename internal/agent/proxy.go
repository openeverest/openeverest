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

// Package agent provides the OpenEverest Kubernetes API gateway used by
// registered spoke clusters.
package agent

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"k8s.io/client-go/rest"
)

var errMissingBearerToken = errors.New("agent bearer token is required")

// NewProxy creates a gateway that authenticates hub requests and proxies only
// Kubernetes API paths. The upstream transport uses the agent's in-cluster
// identity, never the credentials supplied by the hub.
func NewProxy(kubeConfig *rest.Config, bearerToken string) (http.Handler, error) {
	if bearerToken == "" {
		return nil, errMissingBearerToken
	}
	target, err := url.Parse(kubeConfig.Host)
	if err != nil {
		return nil, err
	}
	transport, err := rest.TransportFor(kubeConfig)
	if err != nil {
		return nil, err
	}

	expectedToken := sha256.Sum256([]byte(bearerToken))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Header.Del("Authorization")
		for header := range req.Header {
			if strings.HasPrefix(http.CanonicalHeaderKey(header), "Impersonate-") {
				req.Header.Del(header)
			}
		}
		req.Host = target.Host
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isKubernetesAPIPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		actual := sha256.Sum256([]byte(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")))
		if subtle.ConstantTimeCompare(actual[:], expectedToken[:]) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		proxy.ServeHTTP(w, r)
	}), nil
}

func isKubernetesAPIPath(requestPath string) bool {
	if path.Clean(requestPath) != requestPath {
		return false
	}
	for _, segment := range strings.Split(requestPath, "/") {
		switch segment {
		case "exec", "attach", "portforward", "proxy":
			return false
		}
	}
	for _, prefix := range []string{"/api/", "/apis/", "/openapi/"} {
		if strings.HasPrefix(requestPath, prefix) {
			return true
		}
	}
	return requestPath == "/api" ||
		requestPath == "/apis" ||
		requestPath == "/openapi" ||
		requestPath == "/version"
}
