// Package validation provides the validation handler.
package validation

import (
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

type validateHandler struct {
	log           *zap.SugaredLogger
	next          handlers.Handler
	kubeConnector kubernetes.KubernetesConnector
}

// New returns a new RBAC handler.
//
//nolint:ireturn
func New(
	log *zap.SugaredLogger,
	kubeConnector kubernetes.KubernetesConnector,
) handlers.Handler {
	l := log.With("handler", "validator")
	return &validateHandler{
		log:           l,
		kubeConnector: kubeConnector,
	}
}

// SetNext sets the next handler to call in the chain.
func (h *validateHandler) SetNext(next handlers.Handler) {
	h.next = next
}
