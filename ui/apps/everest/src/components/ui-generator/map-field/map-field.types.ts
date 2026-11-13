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
import type { ComponentGroup, MapComponent } from '../ui-generator.types';

export interface MapFieldProps {
  item: MapComponent;
  name: string;
  // Renders one row's fields; `root` is the row's form path, e.g. `labels.2`.
  children: (row: ComponentGroup, root: string) => ReactNode;
}

export interface RowState {
  disabled: boolean;
  // The row was just added, so its first field takes the focus.
  focus: boolean;
}
