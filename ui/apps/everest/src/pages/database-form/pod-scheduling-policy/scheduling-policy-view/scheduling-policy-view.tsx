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

import { Typography } from '@mui/material';
import { WidgetSummaryProps } from 'components/ui-generator/widget-summary-registry';
import { AffinityGroupList } from '../affinity/affinity-group-list';
import { getComponentAffinityGroups } from '../pod-scheduling-policy.utils';
import { SchedulingPolicyTabs } from '../scheduling-policy-tabs';
import { Messages } from '../pod-scheduling-policy-section.messages';

// Read-only counterpart of PodSchedulingPolicySection: same tabs and rule
// cards, without add / edit / delete.
export const SchedulingPolicyView = ({ value }: WidgetSummaryProps) => {
  const components = getComponentAffinityGroups(value);

  if (components.every(({ groups }) => groups.length === 0)) {
    return (
      <Typography variant="body2" sx={{ color: 'text.secondary' }}>
        {Messages.notConfigured}
      </Typography>
    );
  }

  return (
    <SchedulingPolicyTabs
      tabs={components.map(({ key, groups }) => ({
        key,
        count: groups.length,
      }))}
    >
      {(activeKey) => (
        <AffinityGroupList
          groups={components.find(({ key }) => key === activeKey)?.groups ?? []}
        />
      )}
    </SchedulingPolicyTabs>
  );
};
