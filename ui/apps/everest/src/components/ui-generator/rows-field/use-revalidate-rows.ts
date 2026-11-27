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
import { useFormContext } from 'react-hook-form';
import { isPlainObject } from 'components/ui-generator/utils/object-path';
import type { RowsFieldKind } from 'components/ui-generator/utils/rows-field';

// What the field's built-in rules read from each row: what its entry goes by,
// and whether the row counts as filled. A CEL rule on the field may read the
// rest of the row too.
const toRowSignature = (
  row: unknown,
  { roles, identity }: RowsFieldKind,
  withEveryPart: boolean
) => {
  if (!isPlainObject(row)) return null;
  const parts = roles.map((role) => row[role]);
  return withEveryPart
    ? parts
    : [row[identity], parts.some((part) => part !== '')];
};

// Some issues span rows: a duplicate, the entry count, a row filled without
// what its entry goes by, a CEL rule on the field. Typing in one row validates
// only that row's input, so the whole field is re-validated whenever a row's
// identity changes or a row fills or empties, and on any change of a row when
// a CEL rule may read it.
export const useRevalidateRows = (
  name: string,
  // The rows as the form holds them now.
  rows: unknown,
  kind: RowsFieldKind,
  withEveryPart: boolean
) => {
  const { trigger } = useFormContext();
  const signature = JSON.stringify(
    (Array.isArray(rows) ? rows : []).map((row) =>
      toRowSignature(row, kind, withEveryPart)
    )
  );
  const validatedSignature = useRef(signature);

  useEffect(() => {
    if (validatedSignature.current !== signature) {
      validatedSignature.current = signature;
      trigger(name);
    }
  }, [signature, name, trigger]);
};
