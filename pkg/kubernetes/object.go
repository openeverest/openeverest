// everest
// Copyright (C) 2025 Percona LLC
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

package kubernetes

import (
	"bytes"
	"context"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlSerializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultAPIURIPath  = "/api"
	defaultAPIsURIPath = "/apis"
	objectsBufferSize  = 100
)

// ApplyManifestFile accepts manifest file contents, parses into []runtime.Object
// and applies them against the cluster.
func (k *Kubernetes) ApplyManifestFile(ctx context.Context, fileBytes []byte, namespace string, ignoreObjects ...ctrlclient.Object) error {
	objs, err := k.getObjects(fileBytes)
	if err != nil {
		return err
	}
	for i := range objs {
		o := objs[i]

		// Check if this object should be ignored?
		if slices.ContainsFunc(ignoreObjects, func(ign ctrlclient.Object) bool {
			return o.GetKind() == ign.GetObjectKind().GroupVersionKind().Kind &&
				o.GetName() == ign.GetName() &&
				ign.GetNamespace() == namespace
		}) {
			continue
		}

		if err := k.applyTemplateCustomization(ctx, o, namespace); err != nil {
			return err
		}
		err := k.ApplyObject(o)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *Kubernetes) getObjects(f []byte) ([]*unstructured.Unstructured, error) {
	var objs []*unstructured.Unstructured
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(f), objectsBufferSize)
	var err error
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, _, err := yamlSerializer.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return nil, err
		}

		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, err
		}

		objs = append(objs, &unstructured.Unstructured{Object: unstructuredMap})
	}

	return objs, nil //nolint:nilerr // the decode loop intentionally stops at the first error (typically io.EOF) and returns the objects decoded so far
}

// ApplyObject applies object.
func (k *Kubernetes) ApplyObject(obj runtime.Object) error {
	// Instantiate a new restmapper so we discover any new resources before applying object.
	groupResources, err := restmapper.GetAPIGroupResources(k.getDiscoveryClient())
	if err != nil {
		return err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := mapper.RESTMapping(gk, gvk.Version)
	if err != nil {
		return err
	}
	namespace, name, err := k.retrieveMetaFromObject(obj)
	if err != nil {
		return err
	}
	cli, err := k.resourceClient(mapping.GroupVersionKind.GroupVersion())
	if err != nil {
		return err
	}
	helper := resource.NewHelper(cli, mapping)
	return k.applyObject(helper, namespace, name, obj)
}

func (k *Kubernetes) applyObject(helper *resource.Helper, namespace, name string, obj runtime.Object) error {
	if _, err := helper.Get(namespace, name); err != nil {
		_, err = helper.Create(namespace, false, obj)
		if err != nil {
			return err
		}
	} else {
		_, err = helper.Replace(namespace, name, true, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *Kubernetes) applyTemplateCustomization(ctx context.Context, u *unstructured.Unstructured, namespace string) error {
	if err := unstructured.SetNestedField(u.Object, namespace, "metadata", "namespace"); err != nil {
		return err
	}

	kind, ok, err := unstructured.NestedString(u.Object, "kind")
	if err != nil {
		return err
	}

	if ok && kind == "ClusterRoleBinding" {
		if err := k.updateClusterRoleBinding(u, namespace); err != nil {
			return err
		}
	}
	if ok && kind == "Service" {
		// During installation or upgrading of the everest API Server
		// CLI should keep spec.type untouched to prevent overriding of it.
		if err := k.setEverestServiceType(ctx, u, namespace); err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) updateClusterRoleBinding(u *unstructured.Unstructured, namespace string) error {
	sub, ok, err := unstructured.NestedFieldNoCopy(u.Object, "subjects")
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	subjects, ok := sub.([]any)
	if !ok {
		return nil
	}

	for _, s := range subjects {
		sub, ok := s.(map[string]any)
		if !ok {
			continue
		}

		if err := unstructured.SetNestedField(sub, namespace, "namespace"); err != nil {
			return err
		}
	}
	return unstructured.SetNestedSlice(u.Object, subjects, "subjects")
}

func (k *Kubernetes) setEverestServiceType(ctx context.Context, u *unstructured.Unstructured, namespace string) error {
	s, err := k.GetService(ctx, types.NamespacedName{Namespace: namespace, Name: "everest"})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		return nil
	}
	if s != nil && s.Name != "" {
		if err := unstructured.SetNestedField(u.Object, string(s.Spec.Type), "spec", "type"); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kubernetes) retrieveMetaFromObject(obj runtime.Object) (string, string, error) {
	name, err := meta.NewAccessor().Name(obj)
	if err != nil {
		return "", name, err
	}
	namespace, err := meta.NewAccessor().Namespace(obj)
	if err != nil {
		return namespace, name, err
	}
	if namespace == "" {
		namespace = k.Namespace()
	}
	return namespace, name, nil
}

func (k *Kubernetes) resourceClient(gv schema.GroupVersion) (rest.Interface, error) {
	cfg := k.restConfig
	cfg.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	cfg.GroupVersion = &gv
	if len(gv.Group) == 0 {
		cfg.APIPath = defaultAPIURIPath
	} else {
		cfg.APIPath = defaultAPIsURIPath
	}
	return rest.RESTClientFor(cfg)
}
