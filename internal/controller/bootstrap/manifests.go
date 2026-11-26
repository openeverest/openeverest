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

package bootstrap

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/config/backupstorageclasses"
	"github.com/openeverest/openeverest/v2/config/policy"
)

//go:embed lifecycle_policies.yaml.tmpl
var lifecyclePoliciesTemplate string

// BuiltinManifests returns the cluster-scoped objects this version of
// OpenEverest defines, for an OpenEverest running in namespace: the
// admission policies of config/policy, those on lifecycle provenance, and
// the BackupStorageClasses of config/backupstorageclasses. Each policy comes
// before its binding.
func BuiltinManifests(namespace string) ([]*unstructured.Unstructured, error) {
	policies, err := kustomizeResources(policy.Manifests)
	if err != nil {
		return nil, fmt.Errorf("config/policy: %w", err)
	}
	lifecyclePolicies, err := template.New("lifecycle").Parse(lifecyclePoliciesTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse the lifecycle policies: %w", err)
	}
	var lifecycle bytes.Buffer
	if err := lifecyclePolicies.Execute(&lifecycle, struct{ Namespace string }{namespace}); err != nil {
		return nil, fmt.Errorf("render the lifecycle policies: %w", err)
	}
	lifecycleObjs, err := decodeAll(lifecycle.Bytes())
	if err != nil {
		return nil, fmt.Errorf("lifecycle policies: %w", err)
	}
	classes, err := kustomizeResources(backupstorageclasses.Manifests)
	if err != nil {
		return nil, fmt.Errorf("config/backupstorageclasses: %w", err)
	}
	return append(append(policies, lifecycleObjs...), classes...), nil
}

// kustomizeResources decodes the files the kustomization.yaml of fsys lists,
// so the controller applies what kustomize and the Helm chart, which is
// synced from the same directories, ship.
func kustomizeResources(fsys fs.FS) ([]*unstructured.Unstructured, error) {
	raw, err := fs.ReadFile(fsys, "kustomization.yaml")
	if err != nil {
		return nil, err
	}
	var kustomization struct {
		Resources []string `json:"resources"`
	}
	if err := yaml.Unmarshal(raw, &kustomization); err != nil {
		return nil, fmt.Errorf("kustomization.yaml: %w", err)
	}
	if len(kustomization.Resources) == 0 {
		return nil, errors.New("kustomization.yaml lists no resources")
	}
	var objs []*unstructured.Unstructured
	for _, file := range kustomization.Resources {
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}
		fileObjs, err := decodeAll(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		objs = append(objs, fileObjs...)
	}
	return objs, nil
}

func decodeAll(data []byte) ([]*unstructured.Unstructured, error) {
	dec := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), len(data))
	var objs []*unstructured.Unstructured
	for {
		u := &unstructured.Unstructured{}
		err := dec.Decode(&u.Object)
		if errors.Is(err, io.EOF) {
			return objs, nil
		}
		if err != nil {
			return nil, err
		}
		if len(u.Object) == 0 {
			continue
		}
		if u.GetKind() == "" || u.GetName() == "" {
			return nil, errors.New("a manifest without kind or name")
		}
		objs = append(objs, u)
	}
}
