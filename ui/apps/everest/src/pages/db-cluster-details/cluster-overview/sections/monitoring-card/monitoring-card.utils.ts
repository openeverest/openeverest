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
import { InstanceMonitoringStatus } from 'shared-types/monitoring-destinations.types';
import { describeCondition } from 'utils/monitoring-destinations';
import {
  ENTRY_CONFIGURED_CONDITION,
  ENTRY_READY_CONDITION,
  MONITORING_READY_CONDITION,
  REASON_TEXT,
  STATUS_ONLY_REASONS,
} from './monitoring-card.constants';
import { ConditionStatus, MonitoringOverview } from './monitoring-card.types';

type EntryCondition = NonNullable<
  InstanceMonitoringStatus['conditions']
>[number];

const toConditionStatus = (
  value: ConditionStatus | undefined
): ConditionStatus => value ?? 'Unknown';

// Ready decides the entry's status. Its explanation comes from Configured
// unless the entry is configured and only its destination is not ready.
const explainingCondition = (
  configured: EntryCondition | undefined,
  ready: EntryCondition | undefined
) =>
  configured?.status === 'True' && ready?.status !== 'True'
    ? (ready ?? configured)
    : (configured ?? ready);

const describeEntry = (
  condition: EntryCondition | undefined,
  destination: string
) => {
  if (!condition || STATUS_ONLY_REASONS.includes(condition.reason)) {
    return '';
  }
  return (
    REASON_TEXT[condition.reason]?.(destination) ?? describeCondition(condition)
  );
};

// Spec entries drive the list; one core has not reported on yet is Unknown.
export const getMonitoringOverview = (
  instance: Instance
): MonitoringOverview => {
  const statuses = instance.status?.monitoring?.destinations ?? [];
  const condition = instance.status?.conditions?.find(
    (c) => c.type === MONITORING_READY_CONDITION
  );

  return {
    ready: condition?.status,
    entries: (instance.spec?.monitoring?.destinations ?? []).map((entry) => {
      const status = statuses.find((s) => s.name === entry.name);
      const conditions = status?.conditions;
      const configured = conditions?.find(
        (c) => c.type === ENTRY_CONFIGURED_CONDITION
      );
      const ready = conditions?.find((c) => c.type === ENTRY_READY_CONDITION);
      const destination = entry.destinationRef.name;
      return {
        name: entry.name,
        destination,
        mode: status?.mode,
        ready: toConditionStatus(ready?.status),
        details: describeEntry(
          explainingCondition(configured, ready),
          destination
        ),
      };
    }),
  };
};
