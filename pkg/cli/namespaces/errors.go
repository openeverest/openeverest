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

package namespaces

import (
	"errors"
	"fmt"
)

// nolint:gochecknoglobals
var (
	// ErrNamespaceNotExist appears when the namespace does not exist.
	ErrNamespaceNotExist = errors.New("namespace does not exist")

	// ErrNamespaceNotExist appears when the namespace does not exist.
	NewErrNamespaceNotExist = func(namespace string) error {
		return fmt.Errorf("'%s': %w", namespace, ErrNamespaceNotExist)
	}

	// ErrNamespaceAlreadyExists appears when the namespace already exists.
	ErrNamespaceAlreadyExists = errors.New("namespace already exists")

	// ErrNamespaceAlreadyExists appears when the namespace already exists.
	NewErrNamespaceAlreadyExists = func(namespace string) error {
		return fmt.Errorf("'%s': %w", namespace, ErrNamespaceAlreadyExists)
	}

	// ErrNamespaceNotManagedByEverest appears when the namespace is not managed by Everest.
	ErrNamespaceNotManagedByEverest = errors.New("namespace is not managed by Everest")

	// ErrNamespaceNotManagedByEverest appears when the namespace is not managed by Everest.
	NewErrNamespaceNotManagedByEverest = func(namespace string) error {
		return fmt.Errorf("'%s': %w", namespace, ErrNamespaceNotManagedByEverest)
	}

	// // ErrNamespaceAlreadyManagedByEverest appears when the namespace is already owned by Everest.
	ErrNamespaceAlreadyManagedByEverest = errors.New("namespace already exists and is managed by Everest")

	// ErrNamespaceAlreadyManagedByEverest appears when the namespace is already owned by Everest.
	NewErrNamespaceAlreadyManagedByEverest = func(namespace string) error {
		return fmt.Errorf("'%s': %w", namespace, ErrNamespaceAlreadyManagedByEverest)
	}

	// ErrNamespaceListEmpty appears when the provided list of the namespaces is considered empty.
	ErrNamespaceListEmpty = errors.New("namespace list is empty. Specify at least one namespace")

	// ErrNamespaceReserved appears when some of the provided names are forbidden to use.
	ErrNamespaceReserved = func(ns string) error {
		return fmt.Errorf("'%s' namespace is reserved for Everest internals. Please specify another namespace", ns)
	}

	// ErrOperatorsNotSelected appears when no operators are selected for installation.
	ErrOperatorsNotSelected = errors.New("no operators selected for installation. Minimum one operator must be selected")

	// ErrCannotRemoveOperators appears when user tries to delete operator from namespace.
	ErrCannotRemoveOperators = errors.New("cannot remove operators")

	// ErrNamespaceNotEmpty is returned when the namespace is not empty.
	ErrNamespaceNotEmpty = errors.New("cannot remove namespace with running database clusters")

	// ErrInteractiveModeDisabled is returned when interactive mode is disabled.
	ErrInteractiveModeDisabled = errors.New("interactive mode is disabled")
)
