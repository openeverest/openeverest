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

import type { RenderOverrides } from 'components/ui-generator/ui-generator.types';
import type { MappedFieldProps } from './get-mapped-params';

// What the engine sets on a field it renders itself, like a map row's key: a
// name in place of a visible label, and focus for a row just added. Disabling
// goes through `isDisabled`, like any other reason.
export const applyRenderOverrides = (
  props: MappedFieldProps,
  overrides: RenderOverrides | undefined
): MappedFieldProps => {
  if (!overrides) return props;
  const { accessibleName, autoFocus } = overrides;
  return {
    ...props,
    textFieldProps: {
      ...props.textFieldProps,
      ...(autoFocus && { autoFocus: true }),
      ...(accessibleName && {
        inputProps: {
          ...props.textFieldProps?.inputProps,
          'aria-label': accessibleName,
        },
      }),
    },
  };
};
