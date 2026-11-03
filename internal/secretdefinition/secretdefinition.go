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

// Package secretdefinition resolves the definition an OpenEverest Secret
// follows from its labels and checks the Secret against it.
//
// A Secret names the object that defines its shape (its source) with exactly
// one source label, and the definition within that source with
// openeverest.io/definition:
//
//   - openeverest.io/provider: a key of the Provider's spec.secrets;
//   - openeverest.io/monitoring-class: an authentication mode of the
//     MonitoringClass.
//
// The one exception is the definition core owns, ca-bundle: a CA bundle is
// not a credential of any Provider or class, so it carries
// openeverest.io/definition: ca-bundle and no source label.
package secretdefinition

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// Kind is the kind of object that defines a Secret.
type Kind string

const (
	// KindProvider is a Provider; the definition is a key of its spec.secrets.
	KindProvider Kind = "Provider"
	// KindMonitoringClass is a MonitoringClass; the definition is one of its
	// authentication modes.
	KindMonitoringClass Kind = "MonitoringClass"
	// KindCore is OpenEverest itself, for the definitions no Provider or class
	// owns. A Secret of a core definition carries no source label.
	KindCore Kind = "Core"
)

var (
	// ErrInvalidLabels is returned when a Secret's labels do not name exactly
	// one source and a definition.
	ErrInvalidLabels = errors.New("invalid definition labels")
	// ErrSourceNotFound is returned when the source a Secret names does not exist.
	ErrSourceNotFound = errors.New("definition source not found")
	// ErrInvalidSecret is returned when a Secret does not match its definition.
	ErrInvalidSecret = errors.New("secret does not match its definition")
)

// sourceLabel is a label that names the source of a Secret's definition.
type sourceLabel struct {
	label string
	kind  Kind
}

// sourceLabels returns the source labels, each with the kind it names.
func sourceLabels() []sourceLabel {
	return []sourceLabel{
		{label: common.OpenEverestProviderLabel, kind: KindProvider},
		{label: common.OpenEverestMonitoringClassLabel, kind: KindMonitoringClass},
	}
}

// Getter reads the objects that define Secrets.
type Getter interface {
	GetProvider(ctx context.Context, key ctrlclient.ObjectKey) (*corev1alpha1.Provider, error)
	GetMonitoringClass(ctx context.Context, key ctrlclient.ObjectKey) (*monitoringv1alpha1.MonitoringClass, error)
}

// Ref names the definition a Secret follows.
type Ref struct {
	// Kind is the kind of the source.
	Kind Kind
	// Source is the name of the Provider or MonitoringClass; empty for a core
	// definition.
	Source string
	// Definition is the definition within the source.
	Definition string
}

// RefFromLabels returns the definition the labels name. It fails unless they
// name exactly one source and a definition, or a core definition and no source.
func RefFromLabels(labels map[string]string) (Ref, error) {
	var refs []Ref
	for _, s := range sourceLabels() {
		if v := labels[s.label]; v != "" {
			refs = append(refs, Ref{Kind: s.kind, Source: v})
		}
	}
	definition := labels[common.OpenEverestDefinitionLabel]

	switch {
	case len(refs) == 0 && definition == "":
		return Ref{}, fmt.Errorf("%w: label the Secret with one of %s, and with %s",
			ErrInvalidLabels, sourceLabelNames(), common.OpenEverestDefinitionLabel,
		)
	case len(refs) == 0 && definition == common.OpenEverestCABundleDefinition:
		return Ref{Kind: KindCore, Definition: definition}, nil
	case len(refs) == 0:
		return Ref{}, fmt.Errorf("%w: %s %q needs one of %s; only %q needs none",
			ErrInvalidLabels, common.OpenEverestDefinitionLabel, definition, sourceLabelNames(),
			common.OpenEverestCABundleDefinition,
		)
	case len(refs) > 1:
		return Ref{}, fmt.Errorf("%w: set only one of %s", ErrInvalidLabels, sourceLabelNames())
	case definition == "":
		return Ref{}, fmt.Errorf("%w: missing %s label", ErrInvalidLabels, common.OpenEverestDefinitionLabel)
	}

	ref := refs[0]
	ref.Definition = definition
	return ref, nil
}

// Validate checks secret against the definition its labels name.
//
// Error messages name keys, never values.
func Validate(ctx context.Context, g Getter, secret *corev1.Secret) error {
	// The control plane fills these types with credentials of the cluster.
	if t := secret.Type; t == corev1.SecretTypeServiceAccountToken || t == corev1.SecretTypeBootstrapToken {
		return fmt.Errorf("%w: Secrets of type %q cannot be created", ErrInvalidSecret, t)
	}

	ref, err := RefFromLabels(secret.GetLabels())
	if err != nil {
		return err
	}

	switch ref.Kind {
	case KindProvider:
		return validateProviderSecret(ctx, g, ref, secret)
	case KindMonitoringClass:
		return validateMonitoringClassSecret(ctx, g, ref, secret)
	case KindCore:
		return validateCABundleSecret(secret)
	default:
		return fmt.Errorf("%w: unknown source kind %q", ErrInvalidLabels, ref.Kind)
	}
}

// validateCABundleSecret checks that every key of the Secret holds PEM CA
// certificates and nothing else: a private key pasted along with its
// certificate would otherwise be stored as public trust material.
func validateCABundleSecret(secret *corev1.Secret) error {
	if t := secretType(secret.Type); t != corev1.SecretTypeOpaque {
		return fmt.Errorf("%w: a CA bundle must be of type %q, not %q", ErrInvalidSecret, corev1.SecretTypeOpaque, t)
	}

	data := secretData(secret)
	if len(data) == 0 {
		return fmt.Errorf("%w: a CA bundle needs at least one key", ErrInvalidSecret)
	}
	for _, key := range slices.Sorted(maps.Keys(data)) {
		for rest := data[key]; ; {
			var block *pem.Block
			if block, rest = pem.Decode(rest); block == nil {
				break
			}
			if block.Type != "CERTIFICATE" {
				return fmt.Errorf("%w: key %q holds a %q PEM block; a CA bundle holds only certificates",
					ErrInvalidSecret, key, block.Type,
				)
			}
		}
		if _, err := commonv1alpha1.ParseCABundle(data[key]); err != nil {
			return fmt.Errorf("%w: key %q: %w", ErrInvalidSecret, key, err)
		}
	}
	return nil
}

func validateProviderSecret(ctx context.Context, g Getter, ref Ref, secret *corev1.Secret) error {
	provider, err := g.GetProvider(ctx, ctrlclient.ObjectKey{Name: ref.Source})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("%w: Provider %q", ErrSourceNotFound, ref.Source)
		}
		return fmt.Errorf("get Provider %q: %w", ref.Source, err)
	}

	if err := controller.ValidateSecretSchema(secret, &provider.Spec); err != nil {
		return errors.Join(ErrInvalidSecret, err)
	}
	return nil
}

func validateMonitoringClassSecret(ctx context.Context, g Getter, ref Ref, secret *corev1.Secret) error {
	class, err := g.GetMonitoringClass(ctx, ctrlclient.ObjectKey{Name: ref.Source})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("%w: MonitoringClass %q", ErrSourceNotFound, ref.Source)
		}
		return fmt.Errorf("get MonitoringClass %q: %w", ref.Source, err)
	}

	mode := class.Spec.FindAuthMode(ref.Definition)
	if mode == nil {
		return fmt.Errorf("%w: MonitoringClass %q has no authentication mode %q; it supports %s",
			ErrInvalidSecret, class.Name, ref.Definition, strings.Join(class.Spec.AuthModeNames(), ", "),
		)
	}
	// The reserved modes (WorkloadIdentity, Anonymous) declare no credentials.
	if mode.Credentials == nil {
		return fmt.Errorf("%w: authentication mode %q of MonitoringClass %q takes no Secret",
			ErrInvalidSecret, mode.Name, class.Name,
		)
	}
	if t := secretType(secret.Type); t != corev1.SecretTypeOpaque {
		return fmt.Errorf("%w: credentials of MonitoringClass %q must be of type %q, not %q",
			ErrInvalidSecret, class.Name, corev1.SecretTypeOpaque, t,
		)
	}
	if err := mode.ValidateCredentials(secretData(secret)); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSecret, err)
	}
	return nil
}

// secretType returns t, treating empty as Opaque the way Kubernetes does.
func secretType(t corev1.SecretType) corev1.SecretType {
	if t == "" {
		return corev1.SecretTypeOpaque
	}
	return t
}

// secretData merges data and stringData, stringData taking precedence as it
// does in Kubernetes.
func secretData(secret *corev1.Secret) map[string][]byte {
	data := make(map[string][]byte, len(secret.Data)+len(secret.StringData))
	maps.Copy(data, secret.Data)
	for k, v := range secret.StringData {
		data[k] = []byte(v)
	}
	return data
}

func sourceLabelNames() string {
	labels := sourceLabels()
	names := make([]string, 0, len(labels))
	for _, s := range labels {
		names = append(names, s.label)
	}
	return strings.Join(names, ", ")
}
