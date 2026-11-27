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

import type { z } from 'zod';
import type { ADDED_ROW_FLAG } from './map-field.constants';

// An entry of the map: its key and its value.
interface MapEntry {
  key: string;
  value: string;
}

// One row of a map field in the form; the map in the API is keyed by `key`.
export interface MapRow extends MapEntry {
  // Set on a row added since the form opened, see ADDED_ROW_FLAG.
  [ADDED_ROW_FLAG]?: true;
}

// An entry as a read-only view shows it.
export interface MapDisplayRow extends MapEntry {
  // The key or the value is shown as a mask, not as stored.
  masked?: boolean;
}

// The zod shape of a row's fields, for each kind of row.
export interface MapRowShapes {
  // A row the form was (re)initialised with: a saved entry, a default one.
  openedWith: Record<string, z.ZodTypeAny>;
  // A row the user added since.
  added: Record<string, z.ZodTypeAny>;
}
