package k8s

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/kubernetes"
)

// k8sHandler is usually the last handler in the chain, so it does not have a next handler.
type k8sHandler struct {
	kubeConnector     kubernetes.KubernetesConnector
	log               *zap.SugaredLogger
	versionServiceURL string
	httpClient        *http.Client
}

// New returns a new RBAC handler.
//
//nolint:ireturn
func New(log *zap.SugaredLogger, kubeConnector kubernetes.KubernetesConnector, vsURL string, httpClient *http.Client) handlers.Handler {
	l := log.With("handler", "k8s")
	return &k8sHandler{
		kubeConnector:     kubeConnector,
		log:               l,
		versionServiceURL: vsURL,
		httpClient:        httpClient,
	}
}

// SetNext sets the next handler to call in the chain.
func (h *k8sHandler) SetNext(_ handlers.Handler) {}
