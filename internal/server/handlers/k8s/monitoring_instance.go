package k8s

import (
	"context"
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
)

func (h *k8sHandler) ListMonitoringInstances(ctx context.Context, namespace string) (*everestv1alpha1.MonitoringConfigList, error) {
	return h.kubeConnector.ListMonitoringConfigs(ctx, ctrlclient.InNamespace(namespace))
}

func (h *k8sHandler) CreateMonitoringInstance(ctx context.Context, namespace string, req *api.CreateMonitoringInstanceJSONRequestBody) (*everestv1alpha1.MonitoringConfig, error) {
	m, err := h.kubeConnector.GetMonitoringConfig(ctx, types.NamespacedName{Namespace: namespace, Name: req.Name})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	if m != nil && m.GetName() != "" {
		return nil, k8serrors.NewAlreadyExists(schema.GroupResource{
			Group:    everestv1alpha1.GroupVersion.Group,
			Resource: "monitoringconfigs",
		}, req.Name,
		)
	}
	apiKey, err := h.getPMMApiKey(ctx, req)
	if err != nil {
		return nil, err
	}
	return h.createMonitoringK8sResources(ctx, namespace, req, apiKey)
}

func (h *k8sHandler) DeleteMonitoringInstance(ctx context.Context, namespace, name string) error {
	delMCObj := &everestv1alpha1.MonitoringConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := h.kubeConnector.DeleteMonitoringConfig(ctx, delMCObj); err != nil {
		return err
	}

	delSecObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return h.kubeConnector.DeleteSecret(ctx, delSecObj)
}

func (h *k8sHandler) GetMonitoringInstance(ctx context.Context, namespace, name string) (*everestv1alpha1.MonitoringConfig, error) {
	return h.kubeConnector.GetMonitoringConfig(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

func (h *k8sHandler) UpdateMonitoringInstance(ctx context.Context, namespace, name string, req *api.UpdateMonitoringInstanceJSONRequestBody) (*everestv1alpha1.MonitoringConfig, error) {
	m, err := h.kubeConnector.GetMonitoringConfig(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, err
	}
	var apiKey string
	if req.Pmm != nil && req.Pmm.ApiKey != "" {
		apiKey = req.Pmm.ApiKey
	}
	skipVerifyTLS := !pointer.Get(req.VerifyTLS)
	if req.Pmm != nil && req.Pmm.User != "" && req.Pmm.Password != "" {
		apiKey, err = m.Spec.PMM.CreatePMMApiKey(
			ctx, fmt.Sprintf("everest-%s-%s", name, uuid.NewString()),
			req.Pmm.User, req.Pmm.Password,
			skipVerifyTLS,
		)
		if err != nil {
			return nil, err
		}
	}
	if apiKey != "" {
		_, err := h.kubeConnector.UpdateSecret(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: h.monitoringConfigSecretData(apiKey),
		})
		if err != nil {
			return nil, fmt.Errorf("could not update k8s secret %s", name)
		}
	}
	if req.Url != "" {
		m.Spec.PMM.URL = req.Url
	}
	if req.AllowedNamespaces != nil {
		m.Spec.AllowedNamespaces = *req.AllowedNamespaces
	}
	if req.VerifyTLS != nil {
		m.Spec.VerifyTLS = req.VerifyTLS
	}
	return h.kubeConnector.UpdateMonitoringConfig(ctx, m)
}

func (h *k8sHandler) getPMMApiKey(ctx context.Context, params *api.CreateMonitoringInstanceJSONRequestBody) (string, error) {
	if params.Pmm != nil && params.Pmm.ApiKey != "" {
		return params.Pmm.ApiKey, nil
	}

	h.log.Debug("Getting PMM API key by username and password")
	skipVerifyTLS := !pointer.Get(params.VerifyTLS)
	config := everestv1alpha1.PMMConfig{URL: params.Url}
	return config.CreatePMMApiKey(
		ctx, fmt.Sprintf("everest-%s-%s", params.Name, uuid.NewString()),
		params.Pmm.User, params.Pmm.Password,
		skipVerifyTLS,
	)
}

func (h *k8sHandler) createMonitoringK8sResources(
	c context.Context, namespace string, params *api.CreateMonitoringInstanceJSONRequestBody, apiKey string,
) (*everestv1alpha1.MonitoringConfig, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: h.monitoringConfigSecretData(apiKey),
	}
	if _, err := h.kubeConnector.CreateSecret(c, secret); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = h.kubeConnector.UpdateSecret(c, secret)
			if err != nil {
				return nil, fmt.Errorf("could not update k8s secret %s", params.Name)
			}
		} else {
			return nil, fmt.Errorf("failed creating secret in the Kubernetes cluster")
		}
	}
	created, err := h.kubeConnector.CreateMonitoringConfig(c, &everestv1alpha1.MonitoringConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: namespace,
		},
		Spec: everestv1alpha1.MonitoringConfigSpec{
			Type: everestv1alpha1.MonitoringType(params.Type),
			PMM: everestv1alpha1.PMMConfig{
				URL: params.Url,
			},
			CredentialsSecretName: params.Name,
			VerifyTLS:             params.VerifyTLS,
		},
	})
	if err != nil {
		delObj := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      params.Name,
				Namespace: namespace,
			},
		}
		if dErr := h.kubeConnector.DeleteSecret(c, delObj); dErr != nil {
			return nil, fmt.Errorf("failed cleaning up the secret because failed creating monitoring instance")
		}
		return nil, err
	}

	return created, nil
}

func (h *k8sHandler) monitoringConfigSecretData(apiKey string) map[string]string {
	return map[string]string{
		"apiKey":   apiKey,
		"username": "api_key",
	}
}
