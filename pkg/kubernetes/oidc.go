package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

// UpdateEverestSettings accepts the full list of Everest settings and updates the settings.
func (k *Kubernetes) UpdateEverestSettings(ctx context.Context, settings common.EverestSettings) error {
	configMapData, err := settings.ToMap()
	if err != nil {
		return err
	}

	cm, err := k.GetConfigMap(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestSettingsConfigMapName})
	// This should never happen because the ConfigMap is installed as a part of the Helm chart.
	// We handle this case anyway so the user can continue to use the settings (and related features such as OIDC),
	// but we can no longer prevent Helm from deleting the ConfigMap during upgrades.
	// In such a case, the user needs to manually add the `helm.sh/resource-policy=keep` annotation to the ConfigMap.
	if k8serrors.IsNotFound(err) {
		_, err = k.CreateConfigMap(ctx, &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Namespace: common.SystemNamespace,
				Name:      common.EverestSettingsConfigMapName,
			},
			Data: configMapData,
		})
		return err
	} else if err != nil {
		return err
	}

	cm.Data = configMapData
	_, err = k.UpdateConfigMap(ctx, cm)
	return err
}

// GetEverestSettings returns Everest settings.
func (k *Kubernetes) GetEverestSettings(ctx context.Context) (common.EverestSettings, error) {
	settings := common.EverestSettings{}
	m, err := k.GetConfigMap(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestSettingsConfigMapName})
	if err != nil {
		return settings, err
	}
	err = settings.FromMap(m.Data)
	return settings, err
}
