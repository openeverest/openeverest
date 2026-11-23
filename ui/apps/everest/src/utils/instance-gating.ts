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

import {
  InstanceConditionType,
  InstanceStatusSource,
} from 'shared-types/instance.types';
import {
  getInstanceCondition,
  isInstanceConditionTrue,
  isInstanceDeleting,
} from './instance-status';

// What the UI allows depends on the condition that makes it possible, never on
// the status word or its tone: those are for people.

// Connection details are published once and outlive an upgrade.
export const hasConnectionDetails = (instance?: InstanceStatusSource) =>
  isInstanceConditionTrue(
    instance,
    InstanceConditionType.ConnectionDetailsReady
  );

export const isInstanceRestoring = (instance?: InstanceStatusSource) =>
  isInstanceConditionTrue(instance, InstanceConditionType.Restoring);

// The operations a spec change has to wait for. Converging without a named
// operation is not one of them.
const EDIT_BLOCKING_OPERATIONS = [
  InstanceConditionType.Restoring,
  InstanceConditionType.Upgrading,
  InstanceConditionType.StorageResizing,
];

// An Instance in trouble stays editable: editing it is how it gets fixed.
// One no provider has reported on is held back, as it was while it had no
// phase: every provider report carries Ready. It can still be deleted.
export const areInstanceEditsBlocked = (instance?: InstanceStatusSource) =>
  isInstanceDeleting(instance) ||
  EDIT_BLOCKING_OPERATIONS.some((operation) =>
    isInstanceConditionTrue(instance, operation)
  ) ||
  !getInstanceCondition(instance, InstanceConditionType.Ready);
