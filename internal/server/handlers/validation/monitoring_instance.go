package validation

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest-operator/utils"
	operatorUtils "github.com/percona/everest-operator/utils"
	"github.com/percona/everest/api"
)

// Used monitoring config error
var errDeleteInUseMonitoringConfig = func(namespace, name string) error {
	return fmt.Errorf("monitoring instance='%s' in namespace='%s' is used by some DB cluster and cannot be deleted", name, namespace)
}

func (h *validateHandler) ListMonitoringInstances(ctx context.Context, namespace string) (*everestv1alpha1.MonitoringConfigList, error) {
	return h.next.ListMonitoringInstances(ctx, namespace)
}

func (h *validateHandler) CreateMonitoringInstance(ctx context.Context, namespace string, req *api.CreateMonitoringInstanceJSONRequestBody) (*everestv1alpha1.MonitoringConfig, error) {
	if err := utils.ValidateEverestResourceName(req.Name, "name"); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	if ok := utils.ValidateURL(req.Url); !ok {
		return nil, errors.Join(ErrInvalidRequest, ErrInvalidURL("url"))
	}
	switch req.Type {
	case api.MonitoringInstanceCreateParamsTypePmm:
		if req.Pmm == nil {
			return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("pmm key is required for type %s", req.Type))
		}

		if req.Pmm.ApiKey == "" && (req.Pmm.User == "" || req.Pmm.Password == "") {
			return nil, errors.Join(ErrInvalidRequest, errors.New("pmm.apiKey or pmm.user with pmm.password fields are required"))
		}
	default:
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("monitoring type %s is not supported", req.Type))
	}
	return h.next.CreateMonitoringInstance(ctx, namespace, req)
}

func (h *validateHandler) DeleteMonitoringInstance(ctx context.Context, namespace, name string) error {
	var mc *metav1.PartialObjectMetadata
	var err error
	if mc, err = h.kubeConnector.GetMonitoringConfigMeta(ctx, types.NamespacedName{Namespace: namespace, Name: name}); err != nil {
		return err
	}

	if operatorUtils.IsEverestObjectInUse(mc) {
		// monitoringConfig is used by some DB cluster
		return errors.Join(ErrInvalidRequest, errDeleteInUseMonitoringConfig(namespace, name))
	}
	return h.next.DeleteMonitoringInstance(ctx, namespace, name)
}

func (h *validateHandler) GetMonitoringInstance(ctx context.Context, namespace, name string) (*everestv1alpha1.MonitoringConfig, error) {
	return h.next.GetMonitoringInstance(ctx, namespace, name)
}

func (h *validateHandler) UpdateMonitoringInstance(ctx context.Context, namespace, name string, req *api.UpdateMonitoringInstanceJSONRequestBody) (*everestv1alpha1.MonitoringConfig, error) {
	if req.Url != "" {
		if ok := operatorUtils.ValidateURL(req.Url); !ok {
			err := ErrInvalidURL("url")
			return nil, errors.Join(ErrInvalidRequest, err)
		}
	}

	switch req.Type {
	case "": // nothing to do.
	case api.MonitoringInstanceUpdateParamsTypePmm:
		if req.Pmm == nil {
			return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("pmm key is required for type %s", req.Type))
		}
	default:
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("monitoring type %s is not supported", req.Type))
	}
	return h.next.UpdateMonitoringInstance(ctx, namespace, name, req)
}
