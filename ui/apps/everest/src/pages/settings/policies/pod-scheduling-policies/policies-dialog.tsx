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

import { MenuItem } from '@mui/material';
import { SelectInput, TextInput } from '@percona/ui-lib';
import { useActiveBreakpoint, useDBEnginesForDbEngineTypes } from 'hooks';
import { FormDialog } from 'components/form-dialog';
import { z } from 'zod';
import { DbEngineType } from '@percona/types';
import { PodSchedulingPolicy } from 'shared-types/affinity.types';
import { rfc_123_schema } from 'utils/common-validation';

const schema = (existingPolicies: PodSchedulingPolicy[]) =>
  z.object({
    name: rfc_123_schema({
      fieldName: 'name',
      maxLength: 63,
    }).refine((val) => {
      const isNameTaken = existingPolicies.some(
        (policy) => policy.metadata.name === val
      );
      return !isNameTaken;
    }, 'Policy name already exists'),
    type: z.nativeEnum(DbEngineType).refine((val) => val !== undefined),
  });

type Props = {
  open: boolean;
  policy?: PodSchedulingPolicy;
  existingPolicies?: PodSchedulingPolicy[];
  submitting?: boolean;
  onClose: () => void;
  onSubmit: (data: z.infer<ReturnType<typeof schema>>) => void;
};

const PoliciesDialog = ({
  open,
  policy,
  submitting,
  existingPolicies = [],
  onClose,
  onSubmit,
}: Props) => {
  const [availableDbTypes] = useDBEnginesForDbEngineTypes();
  const isEditing = !!policy;
  const { isMobile } = useActiveBreakpoint();

  return (
    <FormDialog
      isOpen={open}
      closeModal={onClose}
      submitting={submitting}
      headerMessage={isEditing ? 'Edit policy details' : 'Create policy'}
      onSubmit={onSubmit}
      submitMessage={isEditing ? 'Save' : 'Create'}
      schema={schema(existingPolicies)}
      defaultValues={{
        name: policy?.metadata.name || '',
        type:
          policy?.spec.engineType ||
          (availableDbTypes.length ? availableDbTypes[0].type : undefined),
      }}
    >
      <TextInput
        name="name"
        label="Policy name"
        textFieldProps={{ sx: { minHeight: '64px' } }}
        formHelperTextProps={{
          sx: {
            maxWidth: isMobile ? '300px' : '100%',
            whiteSpace: 'normal',
            wordBreak: 'break-word',
            overflowWrap: 'anywhere',
          },
        }}
      />
      <SelectInput
        name="type"
        label="Technology"
        selectFieldProps={{ disabled: isEditing }}
      >
        {availableDbTypes.map((item) => (
          <MenuItem
            data-testid={`add-db-cluster-button-${item.type}`}
            disabled={!item.available}
            key={item.type}
            value={item.type}
          >
            {String(item.type)}
          </MenuItem>
        ))}
      </SelectInput>
    </FormDialog>
  );
};

export default PoliciesDialog;
