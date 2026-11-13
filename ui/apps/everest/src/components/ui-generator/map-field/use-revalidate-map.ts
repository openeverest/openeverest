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

import { useEffect, useRef } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import { isPlainObject } from 'components/ui-generator/utils/object-path';

// What the map's own rules read from each row: its key, and whether the row
// counts as filled.
const toRowSignature = (row: unknown) =>
  isPlainObject(row) ? [row.key, row.key !== '' || row.value !== ''] : null;

// Some issues span rows: a duplicate key, the entry count, a value without a
// key. Typing in one row validates only that row's input, so the whole map is
// re-validated whenever a key changes or a row fills or empties.
export const useRevalidateMap = (name: string) => {
  const { control, trigger } = useFormContext();
  const rows: unknown = useWatch({ control, name });
  const signature = JSON.stringify(
    (Array.isArray(rows) ? rows : []).map(toRowSignature)
  );
  const validatedSignature = useRef(signature);

  useEffect(() => {
    if (validatedSignature.current !== signature) {
      validatedSignature.current = signature;
      trigger(name);
    }
  }, [signature, name, trigger]);
};
