// everest
// Copyright (C) 2023 Percona LLC
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

import { Instance } from 'shared-types/api.types';

export type InstanceCondition = NonNullable<
  NonNullable<Instance['status']>['conditions']
>[number];

// Mirrors the constants in api/core/v1alpha1/instance_types.go: condition
// types aren't in the generated contract.
export const InstanceConditionType = {
  Available: 'Available',
  Ready: 'Ready',
  Stalled: 'Stalled',
  NeedsAttention: 'NeedsAttention',
  Suspended: 'Suspended',
  // Written by core, never by a provider, while nobody attends to the
  // Instance's current spec.
  Unattended: 'Unattended',
  ConnectionDetailsReady: 'ConnectionDetailsReady',
  Restoring: 'Restoring',
  Upgrading: 'Upgrading',
  StorageResizing: 'StorageResizing',
} as const;

// Whether to wait or to act. It comes from condition statuses only, so a
// reason this client has never seen still gets the right tone.
export type InstanceTone =
  | 'ok'
  | 'progress'
  | 'attention'
  | 'error'
  | 'stopped'
  | 'deleting'
  // Nobody can tell: the provider has stopped reporting on the Instance.
  | 'unknown';

export type InstanceAvailability = 'Yes' | 'No' | 'Unknown';

// What the status is read from. Narrower than Instance in the status, so
// anything shaped like one can be displayed, and wider in the metadata: the
// generated ObjectMeta leaves out the generation the API returns.
export interface InstanceStatusSource {
  metadata?: {
    generation?: number;
    deletionTimestamp?: string;
  };
  status?: {
    phase?: string;
    observedGeneration?: number;
    conditions?: {
      type: string;
      status: string;
      reason?: string;
      message?: string;
    }[];
  };
}

export interface InstanceStatusDisplay {
  tone: InstanceTone;
  // The status word as the API spells it: the Ready condition's reason,
  // unless the status describes an older spec.
  reason: string;
  // The status word as a person reads it. everestctl shows the same one.
  label: string;
  available: InstanceAvailability;
  // One sentence about the status word: the Ready condition's message. While
  // stale it is what core says about nobody attending to the Instance, or
  // empty, because the provider's message describes an older spec.
  message: string;
  // The status describes an older spec than the one stored.
  stale: boolean;
}

export interface InstanceProblem {
  reason: string;
  label: string;
  message: string;
  // Whether waiting can still help. While nobody attends to the Instance,
  // nothing is being retried and nothing has given up.
  outlook: 'retrying' | 'stalled' | 'unattended';
}
