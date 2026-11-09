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
  FormMode,
  isSection,
  Section,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSection } from 'components/ui-generator/utils/preprocess/preprocess-section';
import { getDefaultValues } from 'components/ui-generator/utils/default-values';
import { extractInstanceValues } from 'components/ui-generator/utils/default-values/extract-instance-values';
import { buildSectionZodSchema } from 'components/ui-generator/utils/schema-builder';
import { postprocessSchemaData } from 'components/ui-generator/utils/postprocess/postprocess-schema';
import {
  deepMerge,
  isPlainObject,
} from 'components/ui-generator/utils/object-path';
import {
  MonitoringClass,
  MonitoringConditionType,
} from 'shared-types/monitoring-destinations.types';
import { getAuthModeCredentialsSection } from './auth-mode-credentials';

// The class uiSchema groups its forms by what they fill: the destination's
// spec, the Secret of each auth mode and an Instance's monitoring entry.
export interface MonitoringClassUiSchema {
  destination?: Section;
  instance?: Section;
  credentials: Record<string, Section>;
}

// One class form rendered on its own, prepared by the same pipeline stages the
// ui-generator runs on a provider topology.
export interface MonitoringSectionForm {
  sectionKey: string;
  schema: TopologyUISchemas;
  sections: Record<string, Section>;
}

// Modes core resolves without a Secret.
const SECRETLESS_AUTH_MODES = ['Anonymous', 'WorkloadIdentity'];

const toSectionMap = (value: unknown): Record<string, Section> =>
  isPlainObject(value)
    ? Object.fromEntries(
        Object.entries(value).filter((entry): entry is [string, Section] =>
          isSection(entry[1])
        )
      )
    : {};

// A class nobody claims (Accepted is not True) cannot serve a destination.
export const isMonitoringClassAccepted = (monitoringClass: MonitoringClass) =>
  (monitoringClass.status?.conditions ?? []).some(
    (condition) =>
      condition.type === MonitoringConditionType.Accepted &&
      condition.status === 'True'
  );

// Whether any change to an entry of the class restarts the pods of the
// provider's Instances, or of any provider's when none is given.
export const restartsPods = (
  monitoringClass: MonitoringClass,
  provider?: string
) =>
  (monitoringClass.status?.providers ?? []).some(
    (entry) =>
      (!provider || entry.name === provider) &&
      entry.disruption === 'RollingRestart'
  );

export const getMonitoringClassName = (monitoringClass: MonitoringClass) =>
  monitoringClass.metadata?.name ?? '';

export const getMonitoringClassTitle = (monitoringClass: MonitoringClass) =>
  monitoringClass.spec.displayName || getMonitoringClassName(monitoringClass);

export const isSecretBackedAuthMode = (mode: string) =>
  !SECRETLESS_AUTH_MODES.includes(mode);

// The generated CRD types keep uiSchema opaque, so each form is narrowed here.
export const getMonitoringClassUiSchema = (
  monitoringClass: MonitoringClass
): MonitoringClassUiSchema => {
  const uiSchema: unknown = monitoringClass.spec.uiSchema;
  if (!isPlainObject(uiSchema)) {
    return { credentials: {} };
  }
  return {
    destination: isSection(uiSchema.destination)
      ? uiSchema.destination
      : undefined,
    instance: isSection(uiSchema.instance) ? uiSchema.instance : undefined,
    credentials: toSectionMap(uiSchema.credentials),
  };
};

export const toMonitoringSectionForm = (
  sectionKey: string,
  section: Section,
  formMode: FormMode = FormMode.New
): MonitoringSectionForm => {
  const sections = preprocessSection(sectionKey, section, formMode);
  return {
    sectionKey,
    sections,
    schema: { [sectionKey]: { sections } },
  };
};

export const getSectionFormDefaults = (
  form: MonitoringSectionForm
): Record<string, unknown> => getDefaultValues(form.schema, form.sectionKey);

// Saved values over the defaults, in the form's shape (an array's map becomes
// rows). Values the schema doesn't own are kept, so a save doesn't drop them.
export const getSectionFormValues = (
  form: MonitoringSectionForm,
  saved: Record<string, unknown>
): Record<string, unknown> =>
  deepMerge(
    deepMerge(getSectionFormDefaults(form), saved),
    extractInstanceValues(form.sections, saved, FormMode.Edit)
  );

export const getSectionFormZodSchema = (
  form: MonitoringSectionForm,
  formMode: FormMode = FormMode.New
): z.ZodTypeAny =>
  buildSectionZodSchema(form.sectionKey, form.sections, { formMode }).schema;

// Coerces numbers, drops inactive toggleable groups and empty values.
export const postprocessSectionForm = (
  values: Record<string, unknown>,
  form: MonitoringSectionForm
): Record<string, unknown> =>
  postprocessSchemaData(values, {
    schema: form.schema,
    selectedTopology: form.sectionKey,
  });

// The class's credentials form for the mode or, when it ships none, one masked
// input per credential key the mode declares.
export const getCredentialsSection = (
  monitoringClass: MonitoringClass,
  mode: string
): Section | undefined =>
  getAuthModeCredentialsSection(
    monitoringClass.spec.authModes,
    getMonitoringClassUiSchema(monitoringClass).credentials,
    mode
  );
