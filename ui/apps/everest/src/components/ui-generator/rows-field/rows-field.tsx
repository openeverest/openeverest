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

import { useEffect, useState } from 'react';
import {
  get,
  useFieldArray,
  useFormContext,
  useFormState,
  useWatch,
} from 'react-hook-form';
import { Box, Divider, FormHelperText, IconButton, Stack } from '@mui/material';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import { ActionableLabeledContent } from '@percona/ui-lib';
import {
  createAddedRow,
  getRow,
  getRowsFieldKind,
  isAddedRow,
  isOneEmptyRow,
} from 'components/ui-generator/utils/rows-field';
import { useUiGeneratorContext } from '../ui-generator-context';
import { Messages } from './rows-field.messages';
import { GroupType, RowsFieldParams } from '../ui-generator.types';
import { RowLocks, RowsFieldProps } from './rows-field.types';
import { getRowAccess, hasOwnCelRule, toRowAtIndex } from './rows-field.utils';
import { useRevalidateRows } from './use-revalidate-rows';

// Edits a map or a list as rows. Each row is the item template's fields, drawn
// by the UI Generator like any other field, so every param and rule of theirs
// applies. What a schema locks (a disabled row field, `disableRemove`) holds
// for the rows the form opened with: a row the user adds is theirs to fill or
// remove. The field never shows no rows: its values come with one empty row
// while it has no entries, and removing the last row leaves one.
export const RowsField = ({
  item,
  name,
  children: renderRow,
}: RowsFieldProps) => {
  // A schema may leave `fieldParams` out, and a form that skips preprocess
  // gets the field as written.
  const fieldParams: RowsFieldParams = item.fieldParams ?? {};
  const { loadingDefaultsForEdition } = useUiGeneratorContext();
  const locks: RowLocks = {
    disabled: !!loadingDefaultsForEdition || !!fieldParams.disabled,
    disableAdd: !!fieldParams.disableAdd,
    disableRemove: !!fieldParams.disableRemove,
  };
  const kind = getRowsFieldKind(item);
  const row = getRow(item);
  // Stacked inputs of one row sit as close as two rows do, so a line tells the
  // rows apart.
  const stacked = row.groupType !== GroupType.Line;
  const { control } = useFormContext();
  const { errors } = useFormState({ control, name });
  const { fields, append, remove, replace } = useFieldArray({ control, name });
  // `fields` keeps a row as it was added; what is typed in it is in the form.
  const rows: unknown = useWatch({ control, name });
  const onlyEmptyRow = isOneEmptyRow(rows, kind);
  const [focusNewRow, setFocusNewRow] = useState(false);
  useRevalidateRows(name, rows, kind, hasOwnCelRule(item));

  // autoFocus acts once, as the new row mounts. Dropping the flag right after
  // keeps a row that remounts later (indexes shift on a remove) from taking
  // the focus.
  useEffect(() => {
    if (focusNewRow) setFocusNewRow(false);
  }, [focusNewRow]);

  const fieldError = get(errors, name);
  const errorMessage: string | undefined =
    fieldError?.root?.message ?? fieldError?.message;

  return (
    <ActionableLabeledContent
      label={fieldParams.label}
      actionButtonProps={{
        dataTestId: `${name}-add`,
        buttonText: fieldParams.addButtonLabel ?? Messages.add,
        disabled: locks.disabled || locks.disableAdd,
        onClick: () => {
          append(createAddedRow(kind), { shouldFocus: false });
          setFocusNewRow(true);
        },
      }}
    >
      <Stack
        gap={stacked ? 2 : 1}
        divider={stacked ? <Divider flexItem /> : undefined}
      >
        {fields.map((field, index) => {
          const added = isAddedRow(field);
          const access = getRowAccess(locks, { added, onlyEmptyRow });
          return (
            <Stack key={field.id} direction="row" alignItems="center" gap={1}>
              <Box sx={{ flex: 1, minWidth: 0 }}>
                {renderRow(
                  toRowAtIndex(row, index, {
                    disabled: access.disabled,
                    added,
                    focus: focusNewRow && index === fields.length - 1,
                  }),
                  `${name}.${index}`
                )}
              </Box>
              <IconButton
                aria-label={Messages.remove(index + 1)}
                disabled={access.removeDisabled}
                onClick={() =>
                  fields.length > 1
                    ? remove(index)
                    : replace([createAddedRow(kind)])
                }
              >
                <DeleteOutlineOutlinedIcon />
              </IconButton>
            </Stack>
          );
        })}
      </Stack>
      {(errorMessage || fieldParams.helperText) && (
        // The next field's floating label reaches up into the gap below, which
        // would leave the text closer to that field than to its own rows.
        <FormHelperText error={!!errorMessage} sx={{ mb: 1 }}>
          {errorMessage ?? fieldParams.helperText}
        </FormHelperText>
      )}
    </ActionableLabeledContent>
  );
};
