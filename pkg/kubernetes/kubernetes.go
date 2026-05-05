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

// Package kubernetes provides functionality for kubernetes.
package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"go.uber.org/zap"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	enginefeatureseverestv1alpha1 "github.com/percona/everest-operator/api/enginefeatures.everest/v1alpha1"
	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/common"
)

type (
	// ClusterType defines type of cluster.
	ClusterType string
)

const (
	// ClusterTypeUnknown is for unknown type.
	ClusterTypeUnknown ClusterType = "unknown"
	// ClusterTypeMinikube is for minikube.
	ClusterTypeMinikube ClusterType = "minikube"
	// ClusterTypeEKS is for EKS.
	ClusterTypeEKS ClusterType = "eks"
	// ClusterTypeGKE is for GKE.
	ClusterTypeGKE ClusterType = "gke"
	// ClusterTypeOpenShift is for OpenShift.
	ClusterTypeOpenShift ClusterType = "openshift"
	// ClusterTypeGeneric is a generic type.
	ClusterTypeGeneric ClusterType = "generic"

	// OLMNamespace is the namespace where OLM is installed.
	OLMNamespace = "everest-olm"

	openShiftCatalogNamespace = "openshift-marketplace"

	pollInterval = 5 * time.Second
	pollTimeout  = 15 * time.Minute

	deploymentRestartAnnotation = "kubectl.kubernetes.io/restartedAt"

	backoffInterval   = 5 * time.Second
	backoffMaxRetries = 5

	defaultQPSLimit   = 100
	defaultBurstLimit = 150
)

var once sync.Once

// Kubernetes is a client for Kubernetes.
type Kubernetes struct {
	k8sClient  ctrlclient.Client
	l          *zap.SugaredLogger
	restConfig *rest.Config
	kubeconfig string
	// it is required for handling plain runtime.Objects (ApplyManifestFile)
	// WARNING: do not access this field directly, use getDiscoveryClient() instead.
	// This field is lazy initialized because it is not always needed.
	discoveryClient discovery.DiscoveryInterface
}

// Kubeconfig returns the path to the kubeconfig.
// This value is available only if the client was created with New() function.
func (k *Kubernetes) Kubeconfig() string {
	return k.kubeconfig
}

// New returns new Kubernetes object based on provided kubeconfig.
func New(kubeconfigPath string, l *zap.SugaredLogger) (KubernetesConnector, error) {
	home := os.Getenv("HOME")
	path := strings.ReplaceAll(kubeconfigPath, "~", home)
	path = filepath.Clean(path)
	fileData, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not read kubeconfig file"))
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(fileData)
	if err != nil {
		return nil, err
	}

	restConfig.QPS = defaultQPSLimit
	restConfig.Burst = defaultBurstLimit
	k8client, err := ctrlclient.New(restConfig, getKubernetesClientOptions(nil))
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		k8sClient:  k8client,
		l:          l.With("component", "kubernetes"),
		restConfig: restConfig,
		kubeconfig: path,
	}, nil
}

// NewInCluster creates a new kubernetes client using incluster authentication.
func NewInCluster(l *zap.SugaredLogger, ctx context.Context, cacheOptions *cache.Options) (KubernetesConnector, error) {
	restConfig := ctrl.GetConfigOrDie()
	restConfig.QPS = defaultQPSLimit
	restConfig.Burst = defaultBurstLimit

	var k8sCache cache.Cache
	var err error
	if cacheOptions != nil {
		k8sCache, err = cache.New(restConfig, *cacheOptions)
		if err != nil {
			panic(err)
		}
		go func() {
			l.Info("starting incluster client cache")
			if err := k8sCache.Start(ctx); err != nil {
				l.Errorf("error starting incluster client cache: %s", err)
				os.Exit(1)
			}
		}()
	}

	k8sclient, err := ctrlclient.New(restConfig, getKubernetesClientOptions(k8sCache))
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		k8sClient:  k8sclient,
		l:          l.With("component", "kubernetes"),
		restConfig: restConfig,
	}, nil
}

// CreateScheme creates a new runtime.Scheme.
// It registers all necessary types:
// - standard client-go types
// - Everest CRDs
// - OLM CRDs
// - API extensions
func CreateScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(everestv1alpha1.AddToScheme(scheme))
	utilruntime.Must(enginefeatureseverestv1alpha1.AddToScheme(scheme))
	utilruntime.Must(olmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiextv1.AddToScheme(scheme))
	return scheme
}

func getKubernetesClientOptions(cache cache.Cache) ctrlclient.Options {
	var cacheOptions *ctrlclient.CacheOptions
	if cache != nil {
		cacheOptions = &ctrlclient.CacheOptions{
			Reader: cache,
		}
	}

	return ctrlclient.Options{
		Scheme: CreateScheme(),
		Cache:  cacheOptions,
	}
}

func (k *Kubernetes) getDiscoveryClient() discovery.DiscoveryInterface {
	once.Do(func() {
		httpClient, err := rest.HTTPClientFor(k.restConfig)
		if err != nil {
			panic(err)
		}

		k.discoveryClient, err = discovery.NewDiscoveryClientForConfigAndClient(k.restConfig, httpClient)
		if err != nil {
			panic(err)
		}
	})
	return k.discoveryClient
}

// Config returns *rest.Config.
func (k *Kubernetes) Config() *rest.Config {
	return k.restConfig
}

// NewEmpty returns new empty Kubernetes object.
// useful for testing.
func NewEmpty(l *zap.SugaredLogger) *Kubernetes {
	return &Kubernetes{
		l: l.With("component", "kubernetes"),
	}
}

// WithKubernetesClient sets the k8s client.
func (k *Kubernetes) WithKubernetesClient(c ctrlclient.Client) *Kubernetes {
	k.k8sClient = c
	return k
}

// Namespace returns the Everest system namespace.
func (k *Kubernetes) Namespace() string {
	return common.SystemNamespace
}

// GetEverestID returns the ID of the namespace where everest is deployed.
func (k *Kubernetes) GetEverestID(ctx context.Context) (string, error) {
	namespace, err := k.GetNamespace(ctx, types.NamespacedName{Name: k.Namespace()})
	if err != nil {
		return "", errors.Join(err, fmt.Errorf("can't get namespace='%s'", k.Namespace()))
	}

	if namespace == nil {
		return "", fmt.Errorf("can't get namespace='%s'", k.Namespace())
	}
	return string(namespace.UID), nil
}

func (k *Kubernetes) isOpenshift(ctx context.Context) (bool, error) {
	ns, err := k.GetNamespace(ctx, types.NamespacedName{Name: openShiftCatalogNamespace})
	if err != nil {
		return false, ctrlclient.IgnoreNotFound(err)
	}
	return ns != nil, nil
}

// GetClusterType tries to guess the underlying kubernetes cluster based on storage class.
func (k *Kubernetes) GetClusterType(ctx context.Context) (ClusterType, error) {
	if ok, err := k.isOpenshift(ctx); err != nil {
		return ClusterTypeUnknown, err
	} else if ok {
		return ClusterTypeOpenShift, nil
	}

	// For other types, we will check the storage classes.
	storageClasses, err := k.ListStorageClasses(ctx)
	if err != nil {
		return ClusterTypeUnknown, err
	}
	for _, storageClass := range storageClasses.Items {
		if strings.Contains(storageClass.Provisioner, "aws") {
			return ClusterTypeEKS, nil
		}
		if strings.Contains(storageClass.Provisioner, "gke") {
			return ClusterTypeGKE, nil
		}
		if strings.Contains(storageClass.Provisioner, "minikube") ||
			strings.Contains(storageClass.Provisioner, "kubevirt.io/hostpath-provisioner") ||
			strings.Contains(storageClass.Provisioner, "standard") {
			return ClusterTypeMinikube, nil
		}
	}
	return ClusterTypeGeneric, nil
}

func mergeNamespacesEnvVar(str1, str2 string) string {
	ns1 := strings.Split(str1, ",")
	ns2 := strings.Split(str2, ",")
	nsMap := make(map[string]struct{})

	for _, ns := range ns1 {
		if ns == "" {
			continue
		}
		nsMap[ns] = struct{}{}
	}

	for _, ns := range ns2 {
		if ns == "" {
			continue
		}
		nsMap[ns] = struct{}{}
	}

	namespaces := []string{}
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}

	sort.Strings(namespaces)

	return strings.Join(namespaces, ",")
}
