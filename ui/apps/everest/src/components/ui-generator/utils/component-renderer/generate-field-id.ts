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

import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import { getComponentSourcePath } from '../preprocess/normalized-component';

// Uses the 'path' property if available, otherwise generates an ID from the component name
export const generateFieldId = (
  item: Component | ComponentGroup,
  generatedName: string
): string => {
  const sourcePath = getComponentSourcePath(item);
  if (sourcePath) {
    return sourcePath;
  }

  return `g-${generatedName}`;
};
