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

import type { ComponentGroup } from 'components/ui-generator/ui-generator.types';
import { generateFieldId } from '../component-renderer/generate-field-id';
import { joinPath } from '../object-path';
import { walkLeafComponents } from '../schema-walker/schema-walker';

// Form field names for every leaf under a group — same IDs renderComponent uses.
export const collectGroupFieldNames = (
  group: ComponentGroup,
  root?: string
): string[] => {
  const names: string[] = [];
  if (!group.components) {
    return names;
  }

  walkLeafComponents(group.components, ({ component, generatedName }) => {
    names.push(joinPath(root, generateFieldId(component, generatedName)));
  });

  return names;
};
