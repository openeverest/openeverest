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

import { Box, Stack, Typography } from '@mui/material';
import {
  AffinityOperatorValue,
  AffinityPriority,
  AffinityPriorityValue,
} from 'shared-types/affinity.types';
import { getAffinityRuleTypeLabel } from 'utils/db';
import { AffinityGroup } from './affinity-group.types';
import { Messages } from './affinity-rule-editor.messages';

export const GroupSummary = ({ group }: { group: AffinityGroup }) => {
  const priorityLabel =
    group.priority === AffinityPriority.Preferred
      ? `${AffinityPriorityValue[AffinityPriority.Preferred]}${
          group.weight != null ? ` · ${group.weight}` : ''
        }`
      : AffinityPriorityValue[AffinityPriority.Required];
  const meta = group.topologyKey
    ? `${priorityLabel} · ${group.topologyKey}`
    : priorityLabel;

  return (
    <Stack sx={{ width: '100%', gap: 0.25 }}>
      <Stack
        direction="row"
        sx={{ gap: 0.75, alignItems: 'baseline', flexWrap: 'wrap' }}
      >
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {getAffinityRuleTypeLabel(group.type)}
        </Typography>
        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
          {meta}
        </Typography>
      </Stack>
      <Stack
        direction="row"
        sx={{ gap: 0.5, alignItems: 'baseline', flexWrap: 'wrap' }}
      >
        {group.conditions.map((condition, index) => (
          <Box
            key={index}
            component="span"
            sx={{ display: 'inline-flex', gap: 0.5, alignItems: 'baseline' }}
          >
            {index > 0 && (
              <Typography
                component="span"
                variant="body2"
                sx={{ fontWeight: 700, color: 'primary.main' }}
              >
                {Messages.and}
              </Typography>
            )}
            <Typography
              component="span"
              variant="body2"
              sx={{ fontFamily: 'monospace' }}
            >
              {condition.key || '—'}
            </Typography>
            {condition.operator && (
              <Typography
                component="span"
                variant="body2"
                sx={{ fontWeight: 700, color: 'primary.main' }}
              >
                {AffinityOperatorValue[condition.operator]}
              </Typography>
            )}
            {condition.values?.length ? (
              <Typography
                component="span"
                variant="body2"
                sx={{ fontFamily: 'monospace' }}
              >
                {`[${condition.values.join(', ')}]`}
              </Typography>
            ) : null}
          </Box>
        ))}
      </Stack>
    </Stack>
  );
};
