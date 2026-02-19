package k8s

import (
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// k8sHandler is usually the last handler in the chain, so it does not have a next handler.
type k8sHandler struct {
	kubeConnector     kubernetes.KubernetesConnector
	log               *zap.SugaredLogger
	versionServiceURL string
}

// New returns a new RBAC handler.
//
//nolint:ireturn
func New(log *zap.SugaredLogger, kubeConnector kubernetes.KubernetesConnector, vsURL string) handlers.Handler {
	l := log.With("handler", "k8s")
	return &k8sHandler{
		kubeConnector:     kubeConnector,
		log:               l,
		versionServiceURL: vsURL,
	}
}

// SetNext sets the next handler to call in the chain.
func (h *k8sHandler) SetNext(_ handlers.Handler) {}
