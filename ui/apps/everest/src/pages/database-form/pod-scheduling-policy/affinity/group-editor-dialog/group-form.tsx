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

import { Box, Button, Stack, Tooltip, Typography } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useEffect, useRef } from 'react';
import { useFieldArray, useFormContext, useWatch } from 'react-hook-form';
import { AffinityPriority, AffinityType } from 'shared-types/affinity.types';
import {
  PriorityToggle,
  TopologyKeyInput,
  TypeInput,
  WeightInput,
} from 'pages/settings/policies/pod-scheduling-policies/affinity/affinity-form-dialog/affinity-form/fields';
import { AffinityFormFields } from 'pages/settings/policies/pod-scheduling-policies/affinity/affinity-form-dialog/affinity-form/affinity-form.types';
import { AffinityGroup } from '../affinity-group.types';
import { GroupNotes } from '../group-notes';
import { ConditionRow } from './condition-row';
import { emptyCondition } from './group-editor-dialog.utils';
import { Messages } from './group-editor-dialog.messages';

// Flatten RHF's nested touchedFields into dotted field paths.
const collectTouchedPaths = (node: unknown, prefix = ''): string[] => {
  if (node === true) {
    return prefix ? [prefix] : [];
  }
  if (node && typeof node === 'object') {
    return Object.entries(node).flatMap(([key, value]) =>
      collectTouchedPaths(value, prefix ? `${prefix}.${key}` : key)
    );
  }
  return [];
};

export const GroupForm = () => {
  const { control, trigger, formState } = useFormContext();
  const [type, priority, topologyKey, conditions, source] = useWatch({
    name: [
      AffinityFormFields.type,
      AffinityFormFields.priority,
      AffinityFormFields.topologyKey,
      'conditions',
      'source',
    ],
  });
  const group: AffinityGroup = {
    type,
    priority,
    topologyKey,
    conditions,
    source,
  };
  const { fields, append, remove } = useFieldArray({
    control,
    name: 'conditions',
  });

  // Type/priority switch which term-level fields and per-condition rules apply,
  // so re-validate on change — but only fields the user already touched, so
  // untouched required fields aren't flagged red before they're filled.
  const touchedRef = useRef(formState.touchedFields);
  touchedRef.current = formState.touchedFields;
  const didMount = useRef(false);
  useEffect(() => {
    if (!didMount.current) {
      didMount.current = true;
      return;
    }
    const touched = collectTouchedPaths(touchedRef.current);
    if (touched.length) {
      trigger(touched);
    }
  }, [type, priority, trigger]);

  return (
    // useFlexGap keeps child margins controllable; one override cancels ui-lib's
    // baked-in mt so every gap is just the Stack spacing. Every field carries a
    // persistent helper, so an error swaps in for it without changing height.
    <Stack spacing={2} useFlexGap sx={{ '& .MuiFormControl-root': { mt: 0 } }}>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Typography variant="sectionHeading">{Messages.ruleType}</Typography>
        <PriorityToggle
          sx={{ height: '30px', width: '200px', flexShrink: 0 }}
        />
      </Box>
      <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-start' }}>
        <Box sx={{ flex: 1, '& .MuiFormControl-root': { width: '100%' } }}>
          <TypeInput helperText={Messages.typeHelper} sx={{ width: '100%' }} />
        </Box>
        {priority === AffinityPriority.Preferred && (
          <Box sx={{ flex: 1, '& .MuiFormControl-root': { width: '100%' } }}>
            <WeightInput sx={{ width: '100%' }} />
          </Box>
        )}
      </Box>
      {type !== AffinityType.NodeAffinity && (
        <Box sx={{ '& .MuiFormControl-root': { width: '100%' } }}>
          <TopologyKeyInput sx={{ width: '100%' }} />
        </Box>
      )}
      <Box sx={{ mt: 1 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 1 }}>
          <Typography variant="sectionHeading">
            {Messages.conditions}
          </Typography>
          <Tooltip title={Messages.conditionsInfo} placement="right" arrow>
            <InfoOutlinedIcon
              // Focusable so the tooltip opens from the keyboard; the tooltip itself marks focus.
              tabIndex={0}
              aria-label={Messages.conditions}
              sx={{
                width: 18,
                color: 'action.active',
                '&:focus, &:focus-visible': { outline: 'none' },
              }}
            />
          </Tooltip>
        </Box>
        <Box sx={{ '&:not(:empty)': { mb: 1 } }}>
          <GroupNotes group={group} />
        </Box>
        {fields.map((field, index) => (
          <ConditionRow
            key={field.id}
            index={index}
            affinityType={type}
            canRemove={fields.length > 1}
            onRemove={() => remove(index)}
          />
        ))}
        <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 1 }}>
          <Button
            size="small"
            startIcon={<AddIcon />}
            onClick={() => append(emptyCondition())}
          >
            {Messages.addCondition}
          </Button>
        </Box>
      </Box>
    </Stack>
  );
};
