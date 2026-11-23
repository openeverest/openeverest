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
import {
  Component,
  FormMode,
  isWidgetComponent,
  Section,
} from '../../ui-generator.types';
import { getDefaultValues } from '../default-values';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { applyFieldOverrides } from '../field-overrides';
import {
  deepClone,
  deepMerge,
  deleteByPathAndEmptyParents,
  getByPath,
  isPlainObject,
  parsePath,
  toFormId,
} from 'components/ui-generator/utils/object-path';
import {
  getOpenApiConstraintError,
  getOpenApiConstraints,
} from '../openapi-constraints';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import {
  getComponentSourcePath,
  getComponentTargetPaths,
} from '../preprocess/normalized-component';
import { preprocessSection } from '../preprocess/preprocess-section';
import { buildSectionZodSchema } from '../schema-builder';
import { collectAllSchemaPaths, walkLeafComponents } from '../schema-walker';
import { getInactiveToggleablePaths } from '../toggleable/toggleable';
import {
  SectionForm,
  SectionFormContract,
  SectionFormOptions,
} from './section-form.types';

export const toSectionForm = (
  sectionKey: string,
  section: Section,
  { formMode = FormMode.New, overrides = {}, contract }: SectionFormOptions = {}
): SectionForm => {
  const sections = applyFieldOverrides(
    preprocessSection(sectionKey, section, formMode),
    overrides
  );
  return {
    sectionKey,
    formMode,
    sections,
    schema: { [sectionKey]: { sections } },
    contract,
  };
};

export const getSectionFormDefaults = (
  form: SectionForm
): Record<string, unknown> => getDefaultValues(form.schema, form.sectionKey);

// The values to edit a saved object with: what it holds, read the way the
// fields show it (a badge stripped, a quantity in the field's unit, a
// toggleable group on when it holds a value), over the defaults.
export const getSectionFormValues = (
  form: SectionForm,
  saved: Record<string, unknown>
): Record<string, unknown> =>
  deepMerge(
    getSectionFormDefaults(form),
    extractInstanceValues(form.sections, saved, form.formMode)
  );

// Coerces numbers, drops inactive toggleable groups and empty values. A
// quantity left as saved is sent the way it was saved.
export const postprocessSectionForm = (
  form: SectionForm,
  values: Record<string, unknown>,
  saved?: Record<string, unknown>
): Record<string, unknown> =>
  postprocessSchemaData(values, {
    schema: form.schema,
    selectedTopology: form.sectionKey,
    originalData: saved,
  });

// What an edit sends: the saved object with the fields the form owns set to
// the form's values (one left empty or in a switched-off group removed), and
// everything else kept as saved.
export const getSectionFormUpdate = (
  form: SectionForm,
  values: Record<string, unknown>,
  saved: Record<string, unknown>
): Record<string, unknown> => {
  const update = deepClone(saved);
  collectAllSchemaPaths(form.sections).forEach((path) =>
    deleteByPathAndEmptyParents(update, path)
  );
  return deepMerge(update, postprocessSectionForm(form, values, saved));
};

const isFixedField = (component: Component) =>
  !isWidgetComponent(component) &&
  (component.fieldParams.disabled === true ||
    ('readOnly' in component.fieldParams &&
      component.fieldParams.readOnly === true));

const toContractPath = (
  contract: SectionFormContract,
  path: string
): string[] | undefined => {
  const pathParts = parsePath(path);
  if (!contract.path) {
    return pathParts;
  }
  const rootParts = parsePath(contract.path);
  if (
    pathParts.length > rootParts.length &&
    rootParts.every((part, index) => pathParts[index] === part)
  ) {
    return pathParts.slice(rootParts.length);
  }
  return undefined;
};

// Checked on what the API gets, so a field showing 30 beside an "s" badge is
// checked as 30s. A field the user can't change, a widget, and one in a
// switched-off group are left to the server: they send nothing to fix.
const getContractIssues = (
  form: SectionForm,
  contract: SectionFormContract,
  values: Record<string, unknown>
): { path: string[]; message: string }[] => {
  const sent = postprocessSectionForm(form, values);
  const switchedOff = new Set(
    getInactiveToggleablePaths(form.sections, values)
  );
  const issues: { path: string[]; message: string }[] = [];
  walkLeafComponents(
    form.sections[form.sectionKey]?.components ?? {},
    ({ component }) => {
      const name = getComponentSourcePath(component);
      if (!name || isWidgetComponent(component) || isFixedField(component)) {
        return;
      }
      const message = getComponentTargetPaths(component)
        .filter((path) => !switchedOff.has(path))
        .map((path) => {
          const contractPath = toContractPath(contract, path);
          const constraints =
            contractPath &&
            getOpenApiConstraints(contract.schema, contractPath);
          return constraints
            ? getOpenApiConstraintError(getByPath(sent, path), constraints)
            : undefined;
        })
        .find((error) => error !== undefined);
      if (message) {
        // Zod issue paths must match form-state keys (RHF-safe form ids).
        issues.push({ path: toFormId(name).split('.'), message });
      }
    }
  );
  return issues;
};

export const getSectionFormZodSchema = (form: SectionForm): z.ZodTypeAny => {
  const { schema } = buildSectionZodSchema(form.sectionKey, form.sections, {
    formMode: form.formMode,
  });
  const { contract } = form;
  if (!contract) {
    return schema;
  }
  return schema.superRefine((values: unknown, ctx) => {
    if (!isPlainObject(values)) {
      return;
    }
    getContractIssues(form, contract, values).forEach(({ path, message }) =>
      ctx.addIssue({ code: z.ZodIssueCode.custom, path, message })
    );
  });
};
