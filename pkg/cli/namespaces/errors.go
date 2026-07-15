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

package namespaces

import (
	"errors"
	"fmt"
)

var (
	// ErrNamespaceNotExist appears when the namespace does not exist.
	ErrNamespaceNotExist = errors.New("namespace does not exist")

	// ErrNamespaceAlreadyExists appears when the namespace already exists.
	ErrNamespaceAlreadyExists = errors.New("namespace already exists")

	// ErrNamespaceNotManagedByEverest appears when the namespace is not managed by Everest.
	ErrNamespaceNotManagedByEverest = errors.New("namespace is not managed by Everest")

	// ErrNamespaceAlreadyManagedByEverest appears when the namespace is already owned by Everest.
	ErrNamespaceAlreadyManagedByEverest = errors.New("namespace already exists and is managed by Everest")

	// ErrNamespaceListEmpty appears when the provided list of the namespaces is considered empty.
	ErrNamespaceListEmpty = errors.New("namespace list is empty. Specify at least one namespace")

	// ErrOperatorsNotSelected appears when no operators are selected for installation.
	ErrOperatorsNotSelected = errors.New("no operators selected for installation. Minimum one operator must be selected")

	// ErrCannotRemoveOperators appears when user tries to delete operator from namespace.
	ErrCannotRemoveOperators = errors.New("cannot remove operators")

	// ErrNamespaceNotEmpty is returned when the namespace is not empty.
	ErrNamespaceNotEmpty = errors.New("cannot remove namespace with running database clusters")

	// ErrInteractiveModeDisabled is returned when interactive mode is disabled.
	ErrInteractiveModeDisabled = errors.New("interactive mode is disabled")
)

// NewErrNamespaceNotExist returns an error indicating that the given namespace does not exist.
func NewErrNamespaceNotExist(namespace string) error {
	return fmt.Errorf("'%s': %w", namespace, ErrNamespaceNotExist)
}

// NewErrNamespaceAlreadyExists returns an error indicating that the given namespace already exists.
func NewErrNamespaceAlreadyExists(namespace string) error {
	return fmt.Errorf("'%s': %w", namespace, ErrNamespaceAlreadyExists)
}

// NewErrNamespaceNotManagedByEverest returns an error indicating that the given namespace is not managed by Everest.
func NewErrNamespaceNotManagedByEverest(namespace string) error {
	return fmt.Errorf("'%s': %w", namespace, ErrNamespaceNotManagedByEverest)
}

// NewErrNamespaceAlreadyManagedByEverest returns an error indicating that the given namespace is already owned by Everest.
func NewErrNamespaceAlreadyManagedByEverest(namespace string) error {
	return fmt.Errorf("'%s': %w", namespace, ErrNamespaceAlreadyManagedByEverest)
}

// ErrNamespaceReserved returns an error indicating that the given namespace name is reserved for Everest internals.
func ErrNamespaceReserved(ns string) error {
	return fmt.Errorf("'%s' namespace is reserved for Everest internals. Please specify another namespace", ns)
}
