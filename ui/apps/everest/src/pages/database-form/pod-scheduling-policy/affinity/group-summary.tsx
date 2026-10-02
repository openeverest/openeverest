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
} from 'shared-types/affinity.types';
import { getAffinityRuleTypeLabel } from 'utils/db';
import { AffinityGroup } from './affinity-group.types';
import {
  getPassthroughFieldNames,
  matchesNoPods,
} from './affinity-group-converter';
import { Messages } from './affinity-rule-editor.messages';

export const GroupSummary = ({ group }: { group: AffinityGroup }) => {
  const keptFields = getPassthroughFieldNames(group.passthrough);
  // Priority is carried by the surrounding Required / Preferred heading.
  const meta = [
    group.priority === AffinityPriority.Preferred && group.weight != null
      ? Messages.weight(group.weight)
      : '',
    group.topologyKey ?? '',
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <Stack sx={{ width: '100%', gap: 0.25 }}>
      <Stack
        direction="row"
        sx={{ gap: 0.75, alignItems: 'baseline', flexWrap: 'wrap' }}
      >
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {getAffinityRuleTypeLabel(group.type)}
        </Typography>
        {meta && (
          <Typography variant="caption" sx={{ color: 'text.secondary' }}>
            {meta}
          </Typography>
        )}
      </Stack>
      {/* One condition per line, flowing as text so long values wrap to the
          line start instead of hanging under their own column. */}
      {group.conditions.map((condition, index) => (
        <Typography
          key={index}
          variant="body2"
          sx={{ overflowWrap: 'anywhere' }}
        >
          {index > 0 && (
            <Box
              component="span"
              sx={{ fontWeight: 700, color: 'primary.main' }}
            >
              {Messages.and}{' '}
            </Box>
          )}
          <Box component="span" sx={{ fontFamily: 'monospace' }}>
            {condition.key || '—'}
          </Box>
          {condition.operator && (
            <>
              {' '}
              <Box
                component="span"
                sx={{ fontWeight: 700, color: 'primary.main' }}
              >
                {AffinityOperatorValue[condition.operator] ??
                  condition.operator}
              </Box>
            </>
          )}
          {condition.values?.length ? (
            <>
              {' '}
              <Box component="span" sx={{ fontFamily: 'monospace' }}>
                {`[${condition.values.join(', ')}]`}
              </Box>
            </>
          ) : null}
        </Typography>
      ))}
      {keptFields.length > 0 && (
        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
          {Messages.keptFields(keptFields)}
        </Typography>
      )}
      {matchesNoPods(group) && (
        <Typography variant="caption" sx={{ color: 'warning.main' }}>
          {Messages.noLabelSelector}
        </Typography>
      )}
    </Stack>
  );
};
