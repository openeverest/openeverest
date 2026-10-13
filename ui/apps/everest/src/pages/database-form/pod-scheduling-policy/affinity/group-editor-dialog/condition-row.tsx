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

import { Box, IconButton } from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import { useFormContext, useFormState, useWatch } from 'react-hook-form';
import { SegmentedField } from '@percona/ui-lib';
import {
  AffinityType,
  NUMERIC_AFFINITY_OPERATORS,
} from 'shared-types/affinity.types';
import {
  doesAffinityOperatorRequireValues,
  getAffinityOperators,
} from 'utils/db';
import {
  KeyInput,
  OperatorInput,
} from 'pages/settings/policies/pod-scheduling-policies/affinity/affinity-form-dialog/affinity-form/fields';
import { AffinityFormFields } from 'pages/settings/policies/pod-scheduling-policies/affinity/affinity-form-dialog/affinity-form/affinity-form.types';
import { ConditionValuesInput } from './condition-values-input';
import { Messages } from './group-editor-dialog.messages';

// Width reserved for the small remove IconButton, so it lines up with the field.
const REMOVE_COLUMN_WIDTH = 34;
// We can't tint one border segment, so mark the guilty field by its text colour.
const errorTextSx = {
  '& .MuiInputBase-input, & .MuiSelect-select': { color: 'error.main' },
};

interface ConditionRowProps {
  index: number;
  affinityType: AffinityType;
  canRemove: boolean;
  onRemove: () => void;
}

export const ConditionRow = ({
  index,
  affinityType,
  canRemove,
  onRemove,
}: ConditionRowProps) => {
  const namePrefix = `conditions.${index}.`;
  const keyPath = `${namePrefix}${AffinityFormFields.key}`;
  const operatorPath = `${namePrefix}${AffinityFormFields.operator}`;
  const valuesPath = `${namePrefix}${AffinityFormFields.values}`;

  const { control, getFieldState } = useFormContext();
  const operator = useWatch({ control, name: operatorPath });
  const requiresValues = doesAffinityOperatorRequireValues(operator);
  const isNumeric = NUMERIC_AFFINITY_OPERATORS.includes(operator);

  const formState = useFormState({
    control,
    name: [keyPath, operatorPath, valuesPath],
  });
  const fieldStates = {
    key: getFieldState(keyPath, formState),
    operator: getFieldState(operatorPath, formState),
    values: getFieldState(valuesPath, formState),
  };
  // Surface only the first already-engaged error (fix it, the next one shows).
  const shouldShow = (state: ReturnType<typeof getFieldState>) =>
    !!state.error && (formState.isSubmitted || state.isTouched);
  const order: Array<'key' | 'operator' | 'values'> = requiresValues
    ? ['key', 'operator', 'values']
    : ['key', 'operator'];
  const erroredField =
    order.find((field) => shouldShow(fieldStates[field])) ?? null;
  const errorMessage = erroredField
    ? fieldStates[erroredField].error?.message
    : undefined;

  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
      <SegmentedField
        sx={{ flex: 1 }}
        dividers
        error={!!erroredField}
        helperText={errorMessage}
      >
        <KeyInput
          affinityType={affinityType}
          namePrefix={namePrefix}
          placeholder={Messages.keyPlaceholder}
          sx={{
            flex: 3,
            minWidth: 0,
            ...(erroredField === 'key' && errorTextSx),
          }}
        />
        <OperatorInput
          disabled={false}
          namePrefix={namePrefix}
          operators={getAffinityOperators(affinityType)}
          sx={{
            width: 130,
            flexShrink: 0,
            ...(erroredField === 'operator' && errorTextSx),
          }}
        />
        {requiresValues && (
          <ConditionValuesInput
            namePrefix={namePrefix}
            placeholder={
              isNumeric
                ? Messages.numericValuePlaceholder
                : Messages.valuesPlaceholder
            }
            sx={{
              flex: 3,
              minWidth: 0,
              ...(erroredField === 'values' && errorTextSx),
            }}
          />
        )}
      </SegmentedField>
      <Box
        sx={{
          width: REMOVE_COLUMN_WIDTH,
          flexShrink: 0,
          height: 40,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <IconButton
          aria-label={Messages.removeCondition}
          disabled={!canRemove}
          onClick={onRemove}
          size="small"
        >
          <DeleteOutlineIcon fontSize="small" />
        </IconButton>
      </Box>
    </Box>
  );
};
