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

import type { ReactNode } from 'react';
import type { ComponentGroup, RowsComponent } from '../ui-generator.types';

export interface RowsFieldProps {
  item: RowsComponent;
  name: string;
  // Renders one row's fields; `root` is the row's form path, e.g. `labels.2`.
  children: (row: ComponentGroup, root: string) => ReactNode;
}

// What the field's params say about its rows as a whole.
export interface RowLocks {
  // The field is disabled, and every row with it.
  disabled: boolean;
  disableAdd: boolean;
  disableRemove: boolean;
}

// What tells one row from the others for those locks.
export interface RowStanding {
  // The user added the row since the form opened, or it is the empty row a
  // field with no entries opens with.
  added: boolean;
  // It is the field's only row, and nothing is typed in it.
  onlyEmptyRow: boolean;
}

export interface RowAccess {
  disabled: boolean;
  removeDisabled: boolean;
}

export interface RowState {
  // The row's inputs are disabled.
  disabled: boolean;
  // The user added the row since the form opened, so a `disabled` of its own
  // fields doesn't apply to it.
  added: boolean;
  // The row was just added, so its first field takes the focus.
  focus: boolean;
}
