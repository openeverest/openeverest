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

import { KeyValueArray } from '@percona/ui-lib';
import { useUiGeneratorContext } from '../ui-generator-context';
import { ArrayFieldProps } from './array-field.types';

export const ArrayField = ({
  item: { fieldParams, itemTemplate },
  name,
}: ArrayFieldProps) => {
  const { loadingDefaultsForEdition } = useUiGeneratorContext();

  return (
    <KeyValueArray
      name={name}
      label={fieldParams.label}
      helperText={fieldParams.helperText}
      addButtonText={fieldParams.addButtonLabel}
      keyPlaceholder={itemTemplate.fieldParams?.key?.placeholder}
      valuePlaceholder={itemTemplate.fieldParams?.value?.placeholder}
      disabled={!!loadingDefaultsForEdition || !!fieldParams.disabled}
    />
  );
};
