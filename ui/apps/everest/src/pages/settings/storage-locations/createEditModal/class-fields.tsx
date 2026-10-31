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
import { SelectInput, SwitchInput, TextInput } from '@percona/ui-lib';
import { HiddenInput } from 'components/hidden-input';
import { FieldType } from 'components/ui-generator/ui-generator.types';
import { Messages } from '../storage-locations.messages';
import { ClassField } from './create-edit-modal.utils';

type ClassFieldsProps = {
  fields: ClassField[];
  // Form path the field paths are relative to.
  root: string;
  readOnly?: boolean;
  optional?: boolean;
};

export const ClassFields = ({
  fields,
  root,
  readOnly = false,
  optional = false,
}: ClassFieldsProps) => (
  <>
    {fields.map((field) => {
      const name = `${root}.${field.path}`;
      const isRequired = field.required && !optional;

      if (field.uiType === FieldType.Toggle) {
        return (
          <SwitchInput
            key={field.key}
            name={name}
            label={field.label}
            labelCaption={field.helperText}
            controllerProps={{
              name,
              defaultValue: field.defaultValue ?? false,
            }}
            switchFieldProps={{ disabled: readOnly }}
            formControlLabelProps={{ sx: { mt: 2 } }}
          />
        );
      }

      if (field.uiType === FieldType.Select) {
        return (
          <SelectInput
            key={field.key}
            name={name}
            label={field.label}
            helperText={field.helperText}
            isRequired={isRequired}
            controllerProps={{ name, defaultValue: field.defaultValue ?? '' }}
            selectFieldProps={{ disabled: readOnly }}
          >
            {!isRequired && (
              <MenuItem value="">
                <em>{Messages.createEditModal.none}</em>
              </MenuItem>
            )}
            {field.options.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </SelectInput>
        );
      }

      const Input = field.password ? HiddenInput : TextInput;
      return (
        <Input
          key={field.key}
          name={name}
          label={field.label}
          isRequired={isRequired}
          controllerProps={{ defaultValue: field.defaultValue ?? '' }}
          textFieldProps={{
            placeholder: field.placeholder,
            helperText: field.helperText,
            disabled: readOnly,
          }}
        />
      );
    })}
  </>
);
