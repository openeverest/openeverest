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
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/pmm"
)

// nolint:unused
// log is for logging in this package.
var monitoringconfiglog = logf.Log.WithName("monitoringconfig-resource")

// SetupMonitoringConfigWebhookWithManager registers the webhook for MonitoringConfig in the manager.
func SetupMonitoringConfigWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &monitoringv1alpha1.MonitoringConfig{}).
		WithValidator(&MonitoringConfigCustomValidator{
			Client:    mgr.GetClient(),
			apiReader: mgr.GetAPIReader(),
		}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-monitoring-openeverest-io-v1alpha1-monitoringconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=monitoring.openeverest.io,resources=monitoringconfigs,verbs=create;update,versions=v1alpha1,name=vmonitoringconfig-v1alpha1.kb.io,admissionReviewVersions=v1

// MonitoringConfigCustomValidator struct is responsible for validating the MonitoringConfig resource
// when it is created, updated, or deleted.
type MonitoringConfigCustomValidator struct {
	Client client.Client
	// apiReader bypasses the cache and directly reads from the API server.
	apiReader client.Reader
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type MonitoringConfig.
func (v *MonitoringConfigCustomValidator) ValidateCreate(ctx context.Context, obj *monitoringv1alpha1.MonitoringConfig) (admission.Warnings, error) {
	monitoringconfiglog.Info("Validation for MonitoringConfig upon creation", "name", obj.GetName())

	return nil, v.validateMonitoringConfig(ctx, obj)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type MonitoringConfig.
func (v *MonitoringConfigCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *monitoringv1alpha1.MonitoringConfig) (admission.Warnings, error) {
	monitoringconfiglog.Info("Validation for MonitoringConfig upon update", "name", newObj.GetName())

	return nil, v.validateMonitoringConfig(ctx, newObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type MonitoringConfig.
func (v *MonitoringConfigCustomValidator) ValidateDelete(_ context.Context, obj *monitoringv1alpha1.MonitoringConfig) (admission.Warnings, error) {
	return nil, nil
}

// validateMonitoringConfig performs checks secret contains valid PMM API key
// by sening a request to PMM server.
func (v *MonitoringConfigCustomValidator) validateMonitoringConfig(ctx context.Context, mc *monitoringv1alpha1.MonitoringConfig) error {
	if !mc.DeletionTimestamp.IsZero() {
		return nil
	}

	secretName := mc.Spec.CredentialsSecretName
	if secretName == "" {
		return errors.New("missing secret name")
	}

	secret := corev1.Secret{}
	if err := v.apiReader.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: mc.GetNamespace(),
	}, &secret); err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	apiKey, ok := secret.Data["apiKey"]
	if !ok {
		return fmt.Errorf("missing apiKey in the secret %s", secretName)
	}

	var skipVerifyTLS bool
	if mc.Spec.VerifyTLS != nil {
		skipVerifyTLS = !*mc.Spec.VerifyTLS
	}

	_, err := pmm.GetPMMServerVersion(ctx, mc.Spec.PMM.URL, string(apiKey), skipVerifyTLS)
	if err != nil {
		return fmt.Errorf("failed to get PMM server version: %w", err)
	}

	return nil
}
