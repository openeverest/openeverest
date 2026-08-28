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

import { z } from 'zod';
import { FieldType } from 'components/ui-generator/ui-generator.types';

// Runtime schema for the skeleton UIGenerator depends on: topology → sections →
// components/groups, with a known `uiType` on every leaf. Everything else passes
// through, so authors stay free to use any field params.

// Leaf uiTypes, from the renderer's own enum so the validator can't drift.
export const FIELD_UI_TYPES = Object.values(FieldType);

const fieldUiType = z.nativeEnum(FieldType);

// `group` is a container, not a field, so it has no FieldType member.
const groupUiType = z.enum(['group', 'hidden']);

const fieldSchema = z.object({ uiType: fieldUiType }).passthrough();

const groupSchema: z.ZodTypeAny = z.lazy(() =>
  z
    .object({
      uiType: groupUiType,
      components: z.record(componentOrGroupSchema),
    })
    .passthrough()
);

const componentOrGroupSchema: z.ZodTypeAny = z.lazy(() =>
  z.union([groupSchema, fieldSchema])
);

const sectionSchema = z
  .object({
    components: z.record(componentOrGroupSchema),
    label: z.string().optional(),
    description: z.string().optional(),
    componentsOrder: z.array(z.string()).optional(),
  })
  .passthrough();

const topologySchema = z
  .object({
    sections: z.record(sectionSchema),
    sectionsOrder: z.array(z.string()).optional(),
  })
  .passthrough();

export const topologyUISchemasSchema = z.record(topologySchema);
