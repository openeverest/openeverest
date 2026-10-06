package httputil

import (
	"crypto/tls"
	"net/http"
	"time"
)

// Option is a functional option to configure the HTTP client.
type Option func(*options)

type options struct {
	timeout            time.Duration
	insecureSkipVerify bool
	disableKeepAlives  bool
}

// WithTimeout sets the timeout for the HTTP client.
func WithTimeout(t time.Duration) Option {
	return func(o *options) {
		o.timeout = t
	}
}

// WithInsecureSkipVerify controls whether the client verifies the server's certificate chain and host name.
func WithInsecureSkipVerify(skip bool) Option {
	return func(o *options) {
		o.insecureSkipVerify = skip
	}
}

// WithTransient disables HTTP keep-alives. This is critical for clients
// that are short-lived, ensuring that TCP connections are closed immediately
// and no background transport goroutines are leaked.
func WithTransient() Option {
	return func(o *options) {
		o.disableKeepAlives = true
	}
}

// NewClient returns a new HTTP client with the specified options.
// It clones the DefaultTransport to ensure that every returned client has its
// own isolated connection pool and TLS configuration, avoiding data races and test flakiness.
func NewClient(opts ...Option) *http.Client {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	var tr *http.Transport
	if defaultTr, ok := http.DefaultTransport.(*http.Transport); ok {
		tr = defaultTr.Clone()
	} else {
		// Fallback if DefaultTransport was overridden with a different RoundTripper implementation
		tr = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}
	}

	if o.insecureSkipVerify {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{} //nolint:gosec
		}
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	if o.disableKeepAlives {
		tr.DisableKeepAlives = true
	}

	return &http.Client{
		Timeout:   o.timeout,
		Transport: tr,
	}
}
