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

import {
  type Component,
  type ComponentGroup,
  FieldType,
} from '../../ui-generator.types';
import { Messages } from './display-value.messages';

// A schema may leave `fieldParams` out even where the type requires it.
export const isPasswordField = (item: Component | ComponentGroup): boolean =>
  item.uiType === FieldType.Text && item.fieldParams?.type === 'password';

// Read-only views (preview, overview) mask a password field's value, as its
// input does. An empty value stays empty, so it still reads as unset.
export const maskPasswordValue = (
  item: Component | ComponentGroup,
  value: string
): string => (value !== '' && isPasswordField(item) ? Messages.masked : value);
