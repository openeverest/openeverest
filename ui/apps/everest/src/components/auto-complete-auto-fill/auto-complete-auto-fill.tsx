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

import { AutoCompleteInput } from '@percona/ui-lib';
import { useEffect } from 'react';
import { useFormContext } from 'react-hook-form';
import { AutoCompleteAutoFillProps } from './auto-complete-auto-fill.types';

export function AutoCompleteAutoFill<T>({
  name,
  controllerProps,
  label,
  labelProps,
  autoCompleteProps,
  textFieldProps,
  options,
  loading,
  isRequired,
  enableFillFirst = false,
  fillFirstField = 'name',
  disabled,
}: AutoCompleteAutoFillProps<T>) {
  const { setValue, getValues, trigger } = useFormContext();

  useEffect(() => {
    const currentValues = getValues(name);

    if (
      !options?.length ||
      !enableFillFirst ||
      (currentValues !== undefined && currentValues !== null)
    ) {
      return;
    }

    setValue(name, options[0]);
    trigger(name);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [options]);

  return (
    <AutoCompleteInput
      name={name}
      controllerProps={controllerProps}
      label={label}
      labelProps={labelProps}
      loading={loading}
      options={options}
      textFieldProps={textFieldProps}
      autoCompleteProps={{
        isOptionEqualToValue: (option, value) =>
          // @ts-ignore
          option[fillFirstField] === value[fillFirstField],
        getOptionLabel: (option) =>
          // @ts-ignore
          typeof option === 'string' ? option : option[fillFirstField],
        ...autoCompleteProps,
      }}
      isRequired={isRequired}
      disabled={disabled}
    />
  );
}
