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

// The conditions the UI reads by name; the rest are listed in the catalog in
// utils/instance-conditions. Mirrors the constants in
// api/core/v1alpha1/instance_types.go: condition types aren't in the
// generated contract.
export const InstanceConditionType = {
  Available: 'Available',
  Ready: 'Ready',
  Reconciling: 'Reconciling',
  Stalled: 'Stalled',
  NeedsAttention: 'NeedsAttention',
  Suspended: 'Suspended',
  ConnectionDetailsReady: 'ConnectionDetailsReady',
  Restoring: 'Restoring',
} as const;

// Whether to wait or to act, and nothing else: whether the database is up is
// InstanceAvailability.
export type InstanceTone =
  | 'ok'
  | 'progress'
  | 'attention'
  | 'error'
  | 'stopped'
  | 'deleting'
  | 'unknown';

export type InstanceAvailability = 'Yes' | 'No' | 'Unknown';

// What the status function reads. Narrower than Instance on purpose, so that
// anything shaped like one can be displayed, and wider in metadata: the
// generated ObjectMeta leaves out the generation the API returns, and the
// managed fields a Kubernetes API returns.
export interface InstanceStatusSource {
  metadata?: {
    generation?: number;
    creationTimestamp?: string;
    deletionTimestamp?: string;
    managedFields?: { time?: string; subresource?: string }[];
  };
  status?: {
    phase?: string;
    observedGeneration?: number;
    conditions?: InstanceCondition[];
  };
}

// What a person is shown for an Instance's status.
export interface InstanceStatusDisplay {
  tone: InstanceTone;
  // The status word: the Ready condition's reason, unless the status can't
  // be trusted to describe the current spec.
  reason: string;
  label: string;
  available: InstanceAvailability;
  // The Ready condition's message. Empty while stale: the only message
  // there is then describes an older spec.
  message: string;
  // When the state the tone describes began, as the condition reports it.
  since: string | null;
  // Availability and the status word in one line, for the details page.
  sentence: string;
  // The status describes an older spec than the one stored. The status word
  // is then a generic one, and the conditions are what was last observed.
  stale: boolean;
}
