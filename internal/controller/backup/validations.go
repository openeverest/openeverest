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

package backup

import (
	"errors"
	"fmt"
	"sync"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/config/crd/bases"
	"github.com/openeverest/openeverest/v2/internal/classvalidation"
)

// backupStorageCRD is the file of the BackupStorage CRD in bases.CRDs.
const backupStorageCRD = "backup.openeverest.io_backupstorages.yaml"

// storageSpecSchema returns the schema self has in the validations of class:
// BackupStorage.spec as the CRD declares it, with parameters typed by the
// class's parametersSchema.
func storageSpecSchema(class *backupv1alpha1.BackupStorageClass) (*apiextensionsv1.JSONSchemaProps, error) {
	data, err := bases.CRDs.ReadFile(backupStorageCRD)
	if err != nil {
		return nil, err
	}
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(data, crd); err != nil {
		return nil, fmt.Errorf("%s: %w", backupStorageCRD, err)
	}
	if len(crd.Spec.Versions) != 1 || crd.Spec.Versions[0].Schema == nil || crd.Spec.Versions[0].Schema.OpenAPIV3Schema == nil {
		return nil, fmt.Errorf("%s: expected one version with a schema", backupStorageCRD)
	}
	spec, ok := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	if !ok {
		return nil, fmt.Errorf("%s: no spec schema", backupStorageCRD)
	}
	if params := class.Spec.ParametersSchema.OpenAPIV3Schema; params != nil {
		typed := params.DeepCopy()
		if typed.Type == "" {
			typed.Type = "object"
		}
		spec.Properties["parameters"] = *typed
	}
	return &spec, nil
}

// compileValidations compiles the validations of class.
func compileValidations(class *backupv1alpha1.BackupStorageClass) (*classvalidation.Rules, error) {
	if len(class.Spec.Validations) == 0 {
		return &classvalidation.Rules{}, nil
	}
	schema, err := storageSpecSchema(class)
	if err != nil {
		return nil, err
	}
	return classvalidation.Compile(schema, class.Spec.Validations)
}

// validationCache holds the compiled validations of each BackupStorageClass
// at its latest generation, so a storage is not judged by compiling CEL
// again. The zero value is ready to use.
type validationCache struct {
	mu      sync.Mutex
	byClass map[types.UID]compiledValidations
}

type compiledValidations struct {
	generation int64
	rules      *classvalidation.Rules
	err        error
}

// errNoUID is returned for a class without a UID, which is never cached.
var errNoUID = errors.New("no UID")

// rules returns the compiled validations of class, compiling them when the
// class is new or changed.
func (c *validationCache) rules(class *backupv1alpha1.BackupStorageClass) (*classvalidation.Rules, error) {
	if class.UID == "" {
		return nil, errNoUID
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.byClass[class.UID]; ok && cached.generation == class.Generation {
		return cached.rules, cached.err
	}
	rules, err := compileValidations(class)
	if c.byClass == nil {
		c.byClass = make(map[types.UID]compiledValidations)
	}
	c.byClass[class.UID] = compiledValidations{generation: class.Generation, rules: rules, err: err}
	return rules, err
}
