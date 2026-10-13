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

import { Instance, PhaseType } from 'shared-types/api.types';

export const DB_INSTANCE_UNKNOWN_PHASE = 'Unknown' as const;
export type DbInstancePhase =
  | Exclude<PhaseType, undefined>
  | typeof DB_INSTANCE_UNKNOWN_PHASE;

export const DbInstancePhaseStatus: Record<
  string,
  Exclude<PhaseType, undefined>
> = {
  Failed: 'Failed',
  Initializing: 'Initializing',
  Pending: 'Pending',
  Provisioning: 'Provisioning',
  Ready: 'Ready',
  Restoring: 'Restoring',
  Resuming: 'Resuming',
  Suspended: 'Suspended',
  Suspending: 'Suspending',
  Terminating: 'Terminating',
  Updating: 'Updating',
} as const;

export const DbInstancePhaseValues: readonly DbInstancePhase[] = [
  ...Object.values(DbInstancePhaseStatus),
  DB_INSTANCE_UNKNOWN_PHASE,
];

export type InstanceCondition = NonNullable<
  NonNullable<Instance['status']>['conditions']
>[number];

// Mirrors v1alpha1.ConditionPodsScheduled and ConditionPodsReady; condition types aren't in the generated contract.
export const INSTANCE_CONDITION_PODS_SCHEDULED = 'PodsScheduled';
export const INSTANCE_CONDITION_PODS_READY = 'PodsReady';

// NotReady is left out: pods are not ready while they start.
export const PODS_ALERT_REASONS = [
  'Unschedulable',
  'CrashLoopBackOff',
  'ImagePullBackOff',
  'CreateContainerConfigError',
] as const;
export type PodsAlertReason = (typeof PODS_ALERT_REASONS)[number];
export type PodsAlertCondition = InstanceCondition & {
  reason: PodsAlertReason;
};

const isPodsAlertCondition = (c: InstanceCondition): c is PodsAlertCondition =>
  (c.type === INSTANCE_CONDITION_PODS_SCHEDULED ||
    c.type === INSTANCE_CONDITION_PODS_READY) &&
  c.status === 'False' &&
  PODS_ALERT_REASONS.some((reason) => reason === c.reason);

export const getPodsAlertConditions = (
  instance?: Instance
): PodsAlertCondition[] =>
  (instance?.status?.conditions ?? []).filter(isPodsAlertCondition);
