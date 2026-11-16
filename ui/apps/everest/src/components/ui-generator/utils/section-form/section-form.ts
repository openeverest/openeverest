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
import { FormMode, Section } from '../../ui-generator.types';
import { getDefaultValues } from '../default-values';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import { preprocessSection } from '../preprocess/preprocess-section';
import { buildSectionZodSchema } from '../schema-builder';
import { SectionForm } from './section-form.types';

export const toSectionForm = (
  sectionKey: string,
  section: Section,
  formMode: FormMode = FormMode.New
): SectionForm => {
  const sections = preprocessSection(sectionKey, section, formMode);
  return {
    sectionKey,
    formMode,
    sections,
    schema: { [sectionKey]: { sections } },
  };
};

export const getSectionFormDefaults = (
  form: SectionForm
): Record<string, unknown> => getDefaultValues(form.schema, form.sectionKey);

export const getSectionFormZodSchema = (form: SectionForm): z.ZodTypeAny =>
  buildSectionZodSchema(form.sectionKey, form.sections, {
    formMode: form.formMode,
  }).schema;

// Coerces numbers, drops inactive toggleable groups and empty values.
export const postprocessSectionForm = (
  form: SectionForm,
  values: Record<string, unknown>
): Record<string, unknown> =>
  postprocessSchemaData(values, {
    schema: form.schema,
    selectedTopology: form.sectionKey,
  });
