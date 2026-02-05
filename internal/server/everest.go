// everest
// Copyright (C) 2023 Percona LLC
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

// Package server contains the API server implementation.
package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/unrolled/secure"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	clientgo "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/percona/everest/api"
	"github.com/percona/everest/cmd/config"
	"github.com/percona/everest/internal/server/handlers"
	k8shandler "github.com/percona/everest/internal/server/handlers/k8s"
	rbachandler "github.com/percona/everest/internal/server/handlers/rbac"
	valhandler "github.com/percona/everest/internal/server/handlers/validation"
	"github.com/percona/everest/pkg/accounts"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
	"github.com/percona/everest/pkg/oidc"
	"github.com/percona/everest/pkg/session"
	"github.com/percona/everest/public"
)

// EverestServer represents the server struct.
type EverestServer struct {
	config        *config.EverestConfig
	l             *zap.SugaredLogger
	echo          *echo.Echo
	kubeConnector kubernetes.KubernetesConnector
	kubeStreamer  clientgo.Interface
	sessionMgr    *session.Manager
	attemptsStore *RateLimiterMemoryStore
	handler       handlers.Handler
	oidcProvider  *oidc.ProviderConfig
}

func getOIDCProviderConfig(ctx context.Context, kubeClient kubernetes.KubernetesConnector) (*oidc.ProviderConfig, error) {
	settings, err := kubeClient.GetEverestSettings(ctx)
	if client.IgnoreNotFound(err) != nil {
		return nil, errors.Join(err, errors.New("failed to get Everest settings"))
	}

	if settings.OIDCConfigRaw == "" {
		return nil, nil //nolint:nilnil
	}

	oidcConfig, err := settings.OIDCConfig()
	if err != nil {
		return nil, errors.Join(err, errors.New("cannot parse OIDC raw config"))
	}

	oidcProvider, err := oidc.NewProviderConfig(ctx, oidcConfig.IssuerURL)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to create OIDC provider config"))
	}

	return &oidcProvider, nil
}

// NewEverestServer creates and configures everest API.
func NewEverestServer(ctx context.Context, c *config.EverestConfig, l *zap.SugaredLogger) (*EverestServer, error) {
	kubeConnector, err := kubernetes.NewInCluster(l, ctx, nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed creating Kubernetes client"))
	}

	kubeStreamer := clientgo.NewForConfigOrDie(kubeConnector.Config())

	if c.HTTPPort != 0 {
		l.Warn("HTTP_PORT is deprecated, use PORT instead")
		c.ListenPort = c.HTTPPort
	}

	echoServer := echo.New()
	echoServer.Use(echomiddleware.RateLimiter(echomiddleware.NewRateLimiterMemoryStore(rate.Limit(c.APIRequestsRateLimit))))
	middleware, store := sessionRateLimiter(c.CreateSessionRateLimit)
	echoServer.Use(middleware)

	sessionManagerClient, err := createSessionManagerClient(ctx, l)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed creating session manager client"))
	}
	sessMgr, err := session.New(
		ctx, l,
		session.WithAccountManager(sessionManagerClient),
	)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to create session manager"))
	}

	oidcProvider, err := getOIDCProviderConfig(ctx, kubeConnector)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get OIDC provider config"))
	}

	e := &EverestServer{
		config:        c,
		l:             l,
		echo:          echoServer,
		kubeConnector: kubeConnector,
		kubeStreamer:  kubeStreamer,
		sessionMgr:    sessMgr,
		attemptsStore: store,
		oidcProvider:  oidcProvider,
	}
	e.echo.HTTPErrorHandler = e.errorHandlerChain()

	if err := e.setupHandlers(ctx, l, kubeConnector, c.VersionServiceURL); err != nil {
		return nil, err
	}

	if err := e.initHTTPServer(ctx); err != nil {
		return e, err
	}
	return e, err
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// initHTTPServer configures http server for the current EverestServer instance.
func (e *EverestServer) initHTTPServer(ctx context.Context) error {
	// Serve the index.html file.
	indexFS := echo.MustSubFS(public.Index, "dist")
	e.echo.Renderer = &Template{
		templates: template.Must(template.ParseFS(indexFS, "index.html")),
	}
	e.echo.GET("/*", func(c echo.Context) error {
		// Embed the CSP nonce into the template. This nonce was auto-generated
		// for this request and stored in the context by the secure middleware.
		// See the securityHeaders middleware for more information.
		return c.Render(http.StatusOK, "index.html",
			map[string]interface{}{"CSPNonce": secure.CSPNonce(c.Request().Context())},
		)
	}, e.securityHeaders(),
	)

	// Serve static files.
	fsys, err := fs.Sub(public.Static, "dist")
	if err != nil {
		return errors.Join(err, errors.New("error reading filesystem"))
	}
	staticFilesHandler := http.FileServer(http.FS(fsys))
	e.echo.GET("/static/*", echo.WrapHandler(staticFilesHandler), e.securityHeaders())

	// Middlewares
	e.echo.Use(echomiddleware.LoggerWithConfig(echomiddleware.LoggerConfig{
		Format:           echomiddleware.DefaultLoggerConfig.Format,
		CustomTimeFormat: echomiddleware.DefaultLoggerConfig.CustomTimeFormat,
		Skipper: func(c echo.Context) bool {
			return c.Request().RequestURI == "/healthz"
		},
	}))
	e.echo.Pre(echomiddleware.RemoveTrailingSlash())

	// Setup the API handlers.
	swagger, err := api.GetSwagger()
	if err != nil {
		return err
	}
	basePath, err := swagger.Servers.BasePath()
	if err != nil {
		return errors.Join(err, errors.New("could not get base path"))
	}

	apiGroup := e.echo.Group(basePath)

	// Use our validation middleware to check all requests against the OpenAPI schema.
	apiGroup.Use(middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
		SilenceServersWarning: true,
		// This field is required if a security scheme is specified.
		// However, the actual authentication is handled by the JWT middleware, so we can use a noop function here.
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}))

	// Setup and use JWT middleware.
	jwtMW, err := e.jwtMiddleWare(ctx)
	if err != nil {
		return err
	}
	apiGroup.Use(jwtMW)

	blocklistMW, err := e.sessionMgr.BlocklistMiddleWare(newSkipperFunc)
	if err != nil {
		return err
	}
	apiGroup.Use(blocklistMW)

	apiGroup.Use(e.checkOperatorUpgradeState)
	api.RegisterHandlers(apiGroup, e)

	return nil
}

func (e *EverestServer) setupHandlers(
	ctx context.Context,
	log *zap.SugaredLogger,
	kubeConnector kubernetes.KubernetesConnector,
	vsURL string,
) error {
	k8sH := k8shandler.New(log, kubeConnector, vsURL)
	valH := valhandler.New(log, kubeConnector)
	rbacH, err := rbachandler.New(ctx, log, kubeConnector)
	if err != nil {
		return errors.Join(err, errors.New("could not create rbac handler"))
	}
	e.setHandlers(valH, rbacH, k8sH)
	return nil
}

func (e *EverestServer) setHandlers(hs ...handlers.Handler) {
	e.handler = newHandlerChain(hs...)
}

// newHandlerChain chains the handlers in the order they are provided.
func newHandlerChain(hs ...handlers.Handler) handlers.Handler { //nolint:ireturn
	if len(hs) == 0 {
		panic("expected at least one handler")
	}
	if len(hs) == 1 {
		return hs[0]
	}
	for i := 0; i < len(hs)-1; i++ {
		hs[i].SetNext(hs[i+1])
	}
	return hs[0]
}

func (e *EverestServer) newJWTKeyFunc(ctx context.Context) (jwt.Keyfunc, error) {
	var oidcKeyFn jwt.Keyfunc
	if e.oidcProvider != nil {
		fn, err := e.oidcProvider.NewKeyFunc(ctx)
		if err != nil {
			return nil, errors.Join(err, errors.New("failed to get OIDC key function"))
		}
		oidcKeyFn = fn
	}

	return func(token *jwt.Token) (interface{}, error) {
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, errors.New("failed to get claims from token")
		}
		issuer, err := claims.GetIssuer()
		if err != nil {
			return "", errors.Join(err, errors.New("failed to get issuer from claims"))
		}

		if issuer == session.SessionManagerClaimsIssuer {
			return e.sessionMgr.KeyFunc()(token)
		}
		// XXX: currently we use OIDC only, but once we have multiple protocols supported,
		// we should have a way to select which KeyFunc to use.
		if oidcKeyFn != nil {
			return oidcKeyFn(token)
		}
		return nil, errors.New("no key found for token")
	}, nil
}

func (e *EverestServer) jwtMiddleWare(ctx context.Context) (echo.MiddlewareFunc, error) {
	keyFunc, err := e.newJWTKeyFunc(ctx)
	if err != nil {
		return nil, err
	}

	skipper, err := newSkipperFunc()
	if err != nil {
		return nil, err
	}

	tokenLookup := "header:Authorization:Bearer "
	return echojwt.WithConfig(echojwt.Config{
		Skipper:     skipper,
		TokenLookup: tokenLookup,
		KeyFunc:     keyFunc,
		ContextKey:  common.UserCtxKey,
		SuccessHandler: func(c echo.Context) {
			// The user key exists only in the echo.Context object.
			// We will copy it to the context.Context as well.
			ctx := c.Request().Context()
			newCtx := context.WithValue(ctx, common.UserCtxKey, c.Get(common.UserCtxKey))
			newReq := c.Request().WithContext(newCtx)
			c.SetRequest(newReq)
		},
	}), nil
}

func newSkipperFunc() (echomiddleware.Skipper, error) {
	swagger, err := api.GetSwagger()
	if err != nil {
		return nil, err
	}

	// list of API paths to exclude from security checks.
	// Each item is a string in the format of "<method> <path>"
	// For example: ["GET /v1/settings"]
	excluded := []string{}

	for path, pathItem := range swagger.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			// Check if we have explicitly specified that we don't want any security here?
			if operation.Security != nil && len(*operation.Security) == 0 {
				for _, srv := range swagger.Servers {
					excluded = append(excluded, fmt.Sprintf("%s %s", method, srv.URL+path))
				}
			}
		}
	}

	return func(c echo.Context) bool {
		target := c.Request().Method + " " + c.Path()
		return slices.Contains(excluded, target)
	}, nil
}

// Start starts everest server.
func (e *EverestServer) Start(ctx context.Context) error {
	addr := fmt.Sprintf("0.0.0.0:%d", e.config.ListenPort)
	if e.config.TLSCertsPath != "" {
		return e.startHTTPS(ctx, addr)
	}
	return e.echo.Start(addr)
}

func (e *EverestServer) startHTTPS(ctx context.Context, addr string) error {
	// The certwatcher will watch the certificate and key files
	// and reload the certificate if they change.
	watcher, err := certwatcher.New(
		path.Join(e.config.TLSCertsPath, "tls.crt"),
		path.Join(e.config.TLSCertsPath, "tls.key"),
	)
	if err != nil {
		return fmt.Errorf("failed to create cert watcher: %w", err)
	}

	// blocking operation, run in background.
	go func() {
		if err := watcher.Start(ctx); err != nil {
			e.l.Error(errors.Join(err, errors.New("failed to start cert watcher")))
		}
	}()

	e.echo.TLSServer = &http.Server{
		Addr: addr,
		TLSConfig: &tls.Config{
			// server periodically calls GetCertificate and reloads the certificate.
			GetCertificate: watcher.GetCertificate,
		},
	}
	return e.echo.StartServer(e.echo.TLSServer)
}

// Shutdown gracefully stops the Everest server.
func (e *EverestServer) Shutdown(ctx context.Context) error {
	e.l.Info("Shutting down http server")
	if err := e.echo.Shutdown(ctx); err != nil {
		e.l.Error(errors.Join(err, errors.New("could not shut down http server")))
		return err
	}
	e.l.Info("http server shut down")

	return nil
}

func (e *EverestServer) getBodyFromContext(ctx echo.Context, into any) error {
	// GetBody creates a copy of the body to avoid "spoiling" the request before proxing
	reader, err := ctx.Request().GetBody()
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(into); err != nil {
		return errors.Join(err, errors.New("could not decode body"))
	}
	return nil
}

func sessionRateLimiter(limit int) (echo.MiddlewareFunc, *RateLimiterMemoryStore) {
	allButSession := func(c echo.Context) bool {
		return c.Request().URL.Path != "/v1/session"
	}
	config := echomiddleware.DefaultRateLimiterConfig
	config.Skipper = allButSession
	store := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{
		Rate: rate.Limit(limit),
	})
	config.Store = store
	return echomiddleware.RateLimiterWithConfig(config), store
}

func (e *EverestServer) errorHandlerChain() echo.HTTPErrorHandler {
	h := e.echo.DefaultHTTPErrorHandler
	h = everestErrorHandler(h)
	return h
}

func everestErrorHandler(next echo.HTTPErrorHandler) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		echoErrTarget := &echo.HTTPError{}
		switch {
		case errors.As(err, &echoErrTarget):
		case k8serrors.IsNotFound(err):
			statusError := &k8serrors.StatusError{}
			if errors.As(err, &statusError) {
				err = &echo.HTTPError{
					Code:    int(statusError.Status().Code),
					Message: trimWebhookErrorText(statusError.Status().Message),
				}
			}
		case k8serrors.IsForbidden(err),
			k8serrors.IsInvalid(err):
			statusError := &k8serrors.StatusError{}
			if errors.As(err, &statusError) {
				err = &echo.HTTPError{
					Code:    int(statusError.Status().Code),
					Message: trimWebhookErrorText(statusError.Status().Message),
				}
			}
		case k8serrors.IsAlreadyExists(err),
			k8serrors.IsConflict(err):
			err = &echo.HTTPError{
				Code: http.StatusConflict,
			}
		case errors.Is(err, rbachandler.ErrInsufficientPermissions):
			err = &echo.HTTPError{
				Code:    http.StatusForbidden,
				Message: rbachandler.ErrInsufficientPermissions.Error(),
			}
		case errors.Is(err, valhandler.ErrInvalidRequest),
			errors.Is(err, errFailedToReadRequestBody):
			err = &echo.HTTPError{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			}
		default:
			err = &echo.HTTPError{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			}
		}
		next(err, c)
	}
}

func trimWebhookErrorText(fullText string) string {
	monitoringWebhookPrefix := `admission webhook "vmonitoringconfig-v1alpha1.everest.percona.com" denied the request: `
	loadBalancerConfigWebhookPrefix := `admission webhook "vloadbalancerconfig-v1alpha1.everest.percona.com" denied the request: `
	dbcWebhookPrefix := `admission webhook "vdatabasecluster-v1alpha1.everest.percona.com" denied the request: `
	return strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(fullText, loadBalancerConfigWebhookPrefix), monitoringWebhookPrefix), dbcWebhookPrefix)
}

// createSessionManagerClient creates a k8s client for a session manager.
func createSessionManagerClient(ctx context.Context, l *zap.SugaredLogger) (accounts.Interface, error) {
	sessionMgrClientCacheOptions := session.ClientCacheOptions()
	sessionMgrClient, err := kubernetes.NewInCluster(l, ctx, sessionMgrClientCacheOptions)
	if err != nil {
		return nil, err
	}
	return sessionMgrClient.Accounts(), nil
}
