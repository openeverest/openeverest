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
import type { CelExpression } from 'components/ui-generator/ui-generator.types';

export type CelExpValidation = {
  path: string[];
  celExpressions: CelExpression[];
  // Switch name of the toggleable group gating this field; skipped while off.
  activeWhen?: string;
};

// A toggleable child's real schema, enforced at the form root only while its
// group's switch is on.
export type ToggleableFieldRule = {
  switchName: string;
  fieldId: string;
  schema: z.ZodTypeAny;
};

export type ComponentSchemaResult = {
  schemaShape: Record<string, z.ZodTypeAny>;
  celExpValidations: CelExpValidation[];
  celDependencyGroups: string[][];
  toggleableFieldRules: ToggleableFieldRule[];
};
