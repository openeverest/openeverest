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

import z from 'zod';
import type { Section } from 'components/ui-generator/ui-generator.types';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { buildSectionZodSchema } from 'components/ui-generator/utils/schema-builder';

// Validates only the dynamic `pitr` section fields. With no schema, any input
// passes (provider ships no PITR parameters — the modal is not shown anyway).
export const pitrConfigSchema = (sections?: Record<string, Section>) => {
  const base = z.object({}).passthrough();
  if (!sections?.pitr) {
    return base;
  }

  const dynamic = buildSectionZodSchema('pitr', sections, {
    formMode: FormMode.Edit,
  }).schema;

  return base.superRefine((data, ctx) => {
    const result = dynamic.safeParse(data);
    if (!result.success) {
      result.error.issues.forEach((issue) => ctx.addIssue(issue));
    }
  });
};
