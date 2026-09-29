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

import { Stack, Typography } from '@mui/material';
import { Affinity } from 'shared-types/affinity.types';
import { WidgetSummaryProps } from 'components/ui-generator/widget-summary-registry';
import { GroupSummary } from 'pages/database-form/pod-scheduling-policy/affinity/group-summary';
import { affinityToGroups } from 'pages/database-form/pod-scheduling-policy/affinity/affinity-group-converter';
import { Messages } from './affinity-summary.messages';

// The affinity value comes untyped from the instance graph; treat any object as
// an Affinity bag (its keys are optional) and an empty one as "no rules".
const isAffinity = (value: unknown): value is Affinity =>
  typeof value === 'object' && value !== null;

export const AffinitySummary = ({ value }: WidgetSummaryProps) => {
  const groups = affinityToGroups(isAffinity(value) ? value : {});

  if (groups.length === 0) {
    return (
      <Typography variant="body2" sx={{ color: 'text.secondary' }}>
        {Messages.empty}
      </Typography>
    );
  }

  return (
    <Stack spacing={1}>
      {groups.map((group, index) => (
        <GroupSummary key={index} group={group} />
      ))}
    </Stack>
  );
};
