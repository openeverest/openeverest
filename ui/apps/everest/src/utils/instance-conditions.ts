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
  InstanceCondition,
  InstanceConditionType,
  InstanceStatusSource,
} from 'shared-types/instance.types';
import { getReasonLabel } from './instance-status';

// What a condition is for decides how it is read:
// - summary: answers a question about the Instance as a whole;
// - check: True is fine, Unknown is in progress, False is a problem;
// - operation: True while something long is running;
// - notice: True when a decision is wanted and nothing is broken.
export type InstanceConditionCategory =
  | 'summary'
  | 'check'
  | 'operation'
  | 'notice';

export interface InstanceConditionDefinition {
  type: string;
  category: InstanceConditionCategory;
  // A check that has to be True for the Instance to be Ready. One that
  // doesn't is reported without changing the Instance's status.
  gatesReady: boolean;
  title: string;
}

const summary = (type: string, title: string): InstanceConditionDefinition => ({
  type,
  category: 'summary',
  gatesReady: false,
  title,
});

const check = (
  type: string,
  title: string,
  gatesReady = true
): InstanceConditionDefinition => ({
  type,
  category: 'check',
  gatesReady,
  title,
});

const operation = (
  type: string,
  title: string
): InstanceConditionDefinition => ({
  type,
  category: 'operation',
  gatesReady: false,
  title,
});

const notice = (type: string, title: string): InstanceConditionDefinition => ({
  type,
  category: 'notice',
  gatesReady: false,
  title,
});

// The conditions core defines, in the order they are listed to a person.
// Mirrors api/core/v1alpha1/instance_types.go. A type that is not here is
// still shown, as reported, with the other conditions.
export const INSTANCE_CONDITIONS: readonly InstanceConditionDefinition[] = [
  summary(InstanceConditionType.Available, 'Available'),
  summary(InstanceConditionType.Ready, 'Ready'),
  summary(InstanceConditionType.Reconciling, 'Reconciling'),
  summary(InstanceConditionType.Stalled, 'Stalled'),
  summary(InstanceConditionType.NeedsAttention, 'Needs attention'),
  summary(InstanceConditionType.Suspended, 'Stopped'),

  check('ReferencesResolved', 'References resolved'),
  check('Synced', 'Provider in sync'),
  check('PodsScheduled', 'Pods scheduled'),
  check('PodsReady', 'Pods ready'),
  check('EngineHealthy', 'Engine healthy'),
  check('DataSourceReady', 'Data loaded from backup'),
  check('BackupConfigured', 'Backups configured'),
  // An engine exposed only inside the cluster may publish none.
  check(
    InstanceConditionType.ConnectionDetailsReady,
    'Connection details published',
    false
  ),
  // Monitoring never makes a database not Ready.
  check('MonitoringConfigured', 'Monitoring configured', false),
  check('MonitoringReady', 'Monitoring ready', false),

  operation(InstanceConditionType.Restoring, 'Restore'),
  operation('Upgrading', 'Upgrade'),
  operation('StorageResizing', 'Storage resize'),
  operation('Scaling', 'Scaling'),

  notice('MaintenancePending', 'Maintenance pending'),
  notice('ComponentVersionDeprecated', 'Version deprecated'),
];

export interface InstanceConditionEntry {
  condition: InstanceCondition;
  title: string;
  gatesReady: boolean;
}

// An Instance's conditions, arranged the way the Status card presents them.
export interface InstanceStatusDetails {
  // Everything observed to be wrong, the cause of the status first.
  problems: InstanceConditionEntry[];
  passingChecks: InstanceConditionEntry[];
  // Checks that are in progress or failing.
  openChecks: InstanceConditionEntry[];
  runningOperations: InstanceConditionEntry[];
  // Operations the status still remembers the outcome of.
  finishedOperations: InstanceConditionEntry[];
  notices: InstanceConditionEntry[];
  // What a list points at beside the status, because it does not change it:
  // notices, and failing checks the Instance is Ready without.
  markers: InstanceConditionEntry[];
  conditions: InstanceCondition[];
}

// A condition that is absent does not apply to this Instance: it is left
// out, never counted as failing.
export const getInstanceStatusDetails = (
  instance: InstanceStatusSource | undefined
): InstanceStatusDetails => {
  const conditions = instance?.status?.conditions ?? [];
  const entriesOf = (category: InstanceConditionCategory) =>
    INSTANCE_CONDITIONS.filter(
      (definition) => definition.category === category
    ).flatMap((definition): InstanceConditionEntry[] => {
      const condition = conditions.find(
        (item) => item.type === definition.type
      );
      return condition
        ? [
            {
              condition,
              title: definition.title,
              gatesReady: definition.gatesReady,
            },
          ]
        : [];
    });
  const withStatus = (
    entries: InstanceConditionEntry[],
    status: InstanceCondition['status']
  ) => entries.filter((entry) => entry.condition.status === status);

  const checks = entriesOf('check');
  const operations = entriesOf('operation');
  const notices = withStatus(entriesOf('notice'), 'True');
  const failingChecks = withStatus(checks, 'False');

  // Not every problem is a failing check: a rejected spec or a wait that
  // lasted too long is only named by NeedsAttention.
  const attention = conditions.find(
    (item) =>
      item.type === InstanceConditionType.NeedsAttention &&
      item.status === 'True'
  );
  const isCause = (entry: InstanceConditionEntry) =>
    entry.condition.reason === attention?.reason;
  const cause: InstanceConditionEntry[] =
    attention && !failingChecks.some(isCause)
      ? [
          {
            condition: attention,
            title: getReasonLabel(attention.reason),
            gatesReady: true,
          },
        ]
      : [];
  const urgency = (entry: InstanceConditionEntry) =>
    isCause(entry) ? 0 : entry.gatesReady ? 1 : 2;

  return {
    problems: [
      ...cause,
      ...[...failingChecks].sort((a, b) => urgency(a) - urgency(b)),
    ],
    passingChecks: withStatus(checks, 'True'),
    openChecks: checks.filter((entry) => entry.condition.status !== 'True'),
    runningOperations: withStatus(operations, 'True'),
    finishedOperations: withStatus(operations, 'False'),
    notices,
    markers: [
      ...notices,
      ...failingChecks.filter((entry) => !entry.gatesReady),
    ],
    conditions,
  };
};
