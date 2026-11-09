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
  Component,
  FieldType,
  isArrayComponent,
} from 'components/ui-generator/ui-generator.types';
import { objectToRows } from '../array-field';

// A Secret is namespaced, so a cluster-wide schema can't name a default one.
// An array's default is written in the API shape; the form keeps rows.
export const getSchemaDefault = (component: Component): unknown => {
  if (component.uiType === FieldType.Secret || !('fieldParams' in component)) {
    return undefined;
  }
  const defaultValue = component.fieldParams?.defaultValue;
  return isArrayComponent(component) && defaultValue !== undefined
    ? objectToRows(defaultValue)
    : defaultValue;
};
