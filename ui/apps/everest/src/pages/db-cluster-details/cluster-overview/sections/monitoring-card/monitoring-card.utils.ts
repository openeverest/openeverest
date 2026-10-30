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
import { MONITORING_CONFIGURED_CONDITION } from './monitoring-card.constants';
import { ConditionStatus, MonitoringOverview } from './monitoring-card.types';

const describeCondition = ({
  reason,
  message,
}: {
  reason?: string;
  message?: string;
}) => [reason, message].filter(Boolean).join(': ');

const toConditionStatus = (value: string | undefined): ConditionStatus =>
  value === 'True' || value === 'False' ? value : 'Unknown';

// Spec entries drive the list; one core has not reported on yet is Unknown.
export const getMonitoringOverview = (
  instance: Instance
): MonitoringOverview => {
  const statuses = instance.status?.monitoring?.destinations ?? [];
  const condition = instance.status?.conditions?.find(
    (c) => c.type === MONITORING_CONFIGURED_CONDITION
  );

  return {
    configured: condition && {
      status: condition.status,
      details: describeCondition(condition),
    },
    entries: (instance.spec?.monitoring?.destinations ?? []).map((entry) => {
      const status = statuses.find((s) => s.name === entry.name);
      return {
        name: entry.name,
        destination: entry.destinationRef.name,
        mode: status?.mode,
        configured: toConditionStatus(status?.configured),
        details: status ? describeCondition(status) : '',
      };
    }),
  };
};
