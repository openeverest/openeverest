// everest
// Copyright (C) 2025 Percona LLC
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

package validation

import (
	"context"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	operatorUtils "github.com/percona/everest-operator/utils"
	"github.com/percona/everest/api"
	"github.com/percona/everest/pkg/common"
)

var (
	// Invalid engine type error
	errInvalidPSPEngineType = func(engineType everestv1alpha1.EngineType) error {
		return fmt.Errorf("unsupported .spec.engineType='%s'", engineType)
	}
	errUpdatePSPEngineType = errors.New("changing .spec.engineType is forbidden")
	// PXC affinity config errors
	errInvalidPSPAffinityPXCWithPSMDB       = newPspValidationAffinityError(fmt.Sprintf(".spec.affinityConfig.psmdb is not applicable with engineType='%s'", everestv1alpha1.DatabaseEnginePXC))
	errInvalidPSPAffinityPXCWithPostgresql  = newPspValidationAffinityError(fmt.Sprintf(".spec.affinityConfig.postgresql is not applicable with engineType='%s'", everestv1alpha1.DatabaseEnginePXC))
	errInvalidPSPAffinityPXCEmpty           = newPspValidationAffinityError(".spec.affinityConfig.pxc is required")
	errInvalidPSPAffinityPXCComponentsEmpty = newPspValidationAffinityError(".spec.affinityConfig.pxc.engine or .spec.affinityConfig.pxc.proxy is required")
	// PSMDB affinity config errors
	errInvalidPSPAffinityPSMDBWithPXC         = newPspValidationAffinityError(fmt.Sprintf(".spec.affinityConfig.pxc is not applicable with engineType='%s'", everestv1alpha1.DatabaseEnginePSMDB))
	errInvalidPSPAffinityPSMDBWithPostgresql  = newPspValidationAffinityError(fmt.Sprintf(".spec.affinityConfig.postgresql is not applicable with engineType='%s'", everestv1alpha1.DatabaseEnginePSMDB))
	errInvalidPSPAffinityPSMDBEmpty           = newPspValidationAffinityError(".spec.affinityConfig.psmdb is required")
	errInvalidPSPAffinityPSMDBComponentsEmpty = newPspValidationAffinityError(".spec.affinityConfig.psmdb.engine or .spec.affinityConfig.psmdb.proxy or .spec.affinityConfig.psmdb.configServer is required")
	// Postgresql affinity config errors
	errInvalidPSPAffinityPostgresqlWithPXC         = newPspValidationAffinityError(fmt.Sprintf(".spec.affinityConfig.pxc is not applicable with engineType='%s'", everestv1alpha1.DatabaseEnginePostgresql))
	errInvalidPSPAffinityPostgresqlWithPSMDB       = newPspValidationAffinityError(fmt.Sprintf(".spec.affinityConfig.psmdb is not applicable with engineType='%s'", everestv1alpha1.DatabaseEnginePostgresql))
	errInvalidPSPAffinityPostgresqlEmpty           = newPspValidationAffinityError(".spec.affinityConfig.postgresql is required")
	errInvalidPSPAffinityPostgresqlComponentsEmpty = newPspValidationAffinityError(".spec.affinityConfig.postgresql.engine or .spec.affinityConfig.postgresql.proxy is required")
	// Default policies errors
	errUpdateDefaultPSP = func(name string) error {
		return fmt.Errorf("pod scheduling policy with name='%s' is default and cannot be updated", name)
	}
	errDeleteDefaultPSP = func(name string) error {
		return fmt.Errorf("pod scheduling policy with name='%s' is default and cannot be deleted", name)
	}
	// Used policy error
	errDeleteInUsePSP = func(name string) error {
		return fmt.Errorf("pod scheduling policy with name='%s' is used by some DB cluster and cannot be deleted", name)
	}
)

type pspValidationAffinityError struct {
	err error
}

func (e *pspValidationAffinityError) Error() string {
	return fmt.Sprintf("invalid affinity config: %v", e.err)
}

func newPspValidationAffinityError(msg string) *pspValidationAffinityError {
	return &pspValidationAffinityError{err: errors.New(msg)}
}

// CreatePodSchedulingPolicy creates a new pod scheduling policy.
func (h *validateHandler) CreatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error) {
	if err := h.validatePSPCR(psp); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	return h.next.CreatePodSchedulingPolicy(ctx, psp)
}

// UpdatePodSchedulingPolicy updates an existing pod scheduling policy.
func (h *validateHandler) UpdatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error) {
	if err := h.validatePSPCR(psp); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	// validate updated policy params
	if err := h.validatePSPOnUpdate(ctx, psp); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, err
		}
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	return h.next.UpdatePodSchedulingPolicy(ctx, psp)
}

// ListPodSchedulingPolicies lists all pod scheduling policies.
func (h *validateHandler) ListPodSchedulingPolicies(ctx context.Context, params *api.ListPodSchedulingPolicyParams) (*everestv1alpha1.PodSchedulingPolicyList, error) {
	return h.next.ListPodSchedulingPolicies(ctx, params)
}

// DeletePodSchedulingPolicy deletes a pod scheduling policy.
func (h *validateHandler) DeletePodSchedulingPolicy(ctx context.Context, name string) error {
	if err := h.validatePSPOnDelete(ctx, name); err != nil {
		if k8serrors.IsNotFound(err) {
			return err
		}
		return errors.Join(ErrInvalidRequest, err)
	}
	return h.next.DeletePodSchedulingPolicy(ctx, name)
}

// GetPodSchedulingPolicy retrieves a pod scheduling policy by name.
func (h *validateHandler) GetPodSchedulingPolicy(ctx context.Context, name string) (*everestv1alpha1.PodSchedulingPolicy, error) {
	return h.next.GetPodSchedulingPolicy(ctx, name)
}

func (h *validateHandler) validatePSPCR(psp *everestv1alpha1.PodSchedulingPolicy) error {
	if err := operatorUtils.ValidateRFC1035(psp.GetName(), "metadata.name"); err != nil {
		return err
	}

	_, ok := common.OperatorTypeToName[psp.Spec.EngineType]
	if !ok {
		return errInvalidPSPEngineType(psp.Spec.EngineType)
	}

	// The following cases are possible:
	// - policy is created without affinityConfig. In such a case only .metadata.name and .spec.engineType are set.
	// This is exactly how UI behaves - creates an empty policy and later adds affinity rules into it one by one.
	// - policy is created with all required fields filled at once.
	affinityConfig := psp.Spec.AffinityConfig
	if affinityConfig == nil {
		return nil
	}

	// affinityConfig is passed - need to validate it.
	switch psp.Spec.EngineType {
	case everestv1alpha1.DatabaseEnginePXC:
		if affinityConfig.PSMDB != nil {
			// .spec.affinityConfig.psmdb shall not be set with engineType=pxc
			return errInvalidPSPAffinityPXCWithPSMDB
		}
		if affinityConfig.PostgreSQL != nil {
			// .spec.affinityConfig.postgresql shall not be set with engineType=pxc
			return errInvalidPSPAffinityPXCWithPostgresql
		}
		if affinityConfig.PXC == nil {
			return errInvalidPSPAffinityPXCEmpty
		}
		if affinityConfig.PXC.Engine == nil && affinityConfig.PXC.Proxy == nil {
			return errInvalidPSPAffinityPXCComponentsEmpty
		}
	case everestv1alpha1.DatabaseEnginePSMDB:
		if affinityConfig.PXC != nil {
			// .spec.affinityConfig.pxc shall not be set with engineType=psmdb
			return errInvalidPSPAffinityPSMDBWithPXC
		}
		if affinityConfig.PostgreSQL != nil {
			// .spec.affinityConfig.postgresql shall not be set with engineType=psmdb
			return errInvalidPSPAffinityPSMDBWithPostgresql
		}
		if affinityConfig.PSMDB == nil {
			return errInvalidPSPAffinityPSMDBEmpty
		}
		if affinityConfig.PSMDB.Engine == nil && affinityConfig.PSMDB.Proxy == nil && affinityConfig.PSMDB.ConfigServer == nil {
			return errInvalidPSPAffinityPSMDBComponentsEmpty
		}
	case everestv1alpha1.DatabaseEnginePostgresql:
		if affinityConfig.PXC != nil {
			// .spec.affinityConfig.pxc shall not be set with engineType=postgresql
			return errInvalidPSPAffinityPostgresqlWithPXC
		}
		if affinityConfig.PSMDB != nil {
			// .spec.affinityConfig.psmdb shall not be set with engineType=postgresql
			return errInvalidPSPAffinityPostgresqlWithPSMDB
		}
		if affinityConfig.PostgreSQL == nil {
			return errInvalidPSPAffinityPostgresqlEmpty
		}
		if affinityConfig.PostgreSQL.Engine == nil && affinityConfig.PostgreSQL.Proxy == nil {
			return errInvalidPSPAffinityPostgresqlComponentsEmpty
		}
	}
	return nil
}

func (h *validateHandler) validatePSPOnUpdate(ctx context.Context, newPsp *everestv1alpha1.PodSchedulingPolicy) error {
	var oldPsp *everestv1alpha1.PodSchedulingPolicy
	var err error
	if oldPsp, err = h.kubeConnector.GetPodSchedulingPolicy(ctx, types.NamespacedName{Name: newPsp.GetName()}); err != nil {
		return err
	}

	if operatorUtils.IsEverestReadOnlyObject(oldPsp) {
		// default policy update is not allowed
		return errUpdateDefaultPSP(newPsp.GetName())
	}

	if oldPsp.Spec.EngineType != newPsp.Spec.EngineType {
		// changing .spec.engineType is not allowed
		return errUpdatePSPEngineType
	}
	return nil
}

func (h *validateHandler) validatePSPOnDelete(ctx context.Context, pspName string) error {
	var psp *metav1.PartialObjectMetadata
	var err error
	if psp, err = h.kubeConnector.GetPodSchedulingPolicyMeta(ctx, types.NamespacedName{Name: pspName}); err != nil {
		return err
	}

	if operatorUtils.IsEverestReadOnlyObject(psp) {
		// default policy deletion is not allowed
		return errDeleteDefaultPSP(psp.GetName())
	}

	if operatorUtils.IsEverestObjectInUse(psp) {
		// policy is used by some DB cluster
		return errDeleteInUsePSP(psp.GetName())
	}
	return nil
}
