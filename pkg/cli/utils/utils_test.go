package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestCheckHelmInstallation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		everestVersion string
		wantErr        bool
		getDpFunc      func(v string) *appsv1.Deployment
	}{
		{
			everestVersion: "1.4.0",
			wantErr:        false,
			getDpFunc:      getNewDeployment,
		},
		{
			everestVersion: "1.4.0-rc3",
			wantErr:        false,
			getDpFunc:      getNewDeployment,
		},
		{
			everestVersion: "1.3.0",
			wantErr:        true,
			getDpFunc:      getLegacyDeployment,
		},
		{
			everestVersion: "0.0.0",
			wantErr:        false,
			getDpFunc:      getNewDeployment,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		mockClient := fakeclient.NewClientBuilder().
			WithScheme(kubernetes.CreateScheme()).
			WithObjects(tc.getDpFunc(tc.everestVersion))
		k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())

		v, err := CheckHelmInstallation(ctx, k)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tc.everestVersion, v)
		}
	}
}

func getNewDeployment(v string) *appsv1.Deployment {
	return getDeployment(v, common.PerconaEverestDeploymentName)
}

func getLegacyDeployment(v string) *appsv1.Deployment {
	return getDeployment(v, common.PerconaEverestDeploymentNameLegacy)
}

func getDeployment(v, depName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      depName,
			Namespace: common.SystemNamespace,
			CreationTimestamp: metav1.Time{
				Time: metav1.Now().Add(-5),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  common.EverestContainerNameInDeployment,
							Image: "percona/everest:v" + v,
						},
					},
				},
			},
		},
	}
}
