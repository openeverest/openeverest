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

import React, { useCallback, useMemo } from 'react';
import { FormEngineConfig, FormEngineResult, StepDefinition } from './types';
import { useUiGenerator } from '../hooks/ui-generator';
import { getDefaultValues } from '../utils/default-values';
import { postprocessSchemaData } from '../utils/postprocess/postprocess-schema';
import { getSectionStepId } from '../utils/section-step-id';
import { UIGenerator } from '../ui-generator';

/**
 * Core form-engine hook.
 *
 * Takes a preprocessed UI schema + topology and produces everything a
 * multi-step wizard needs: ordered step definitions, Zod validation,
 * default values, field → step mapping, and a submit post-processor.
 *
 * This is the single orchestration point that replaces the scattered
 * logic previously spread across database-form.tsx, useSteps, and
 * several intermediate hooks.
 */
export const useFormEngine = (config: FormEngineConfig): FormEngineResult => {
  const {
    uiSchema,
    selectedTopology,
    staticSteps = [],
    providerObject,
    namespace,
  } = config;

  // 1. Schema processing (sections, zod, field map)
  const stableUiSchema = useMemo(() => uiSchema || {}, [uiSchema]);

  const {
    sections,
    sectionsOrder,
    zodSchema: { schema: zodSchema, celDependencyGroups },
    sectionFieldMap,
  } = useUiGenerator(stableUiSchema, selectedTopology);

  // 2. Build generated steps from schema sections
  const sectionKeys = useMemo(
    () => sectionsOrder || Object.keys(sections),
    [sectionsOrder, sections]
  );

  const generatedSteps: StepDefinition[] = useMemo(
    () =>
      sectionKeys.map((sectionKey): StepDefinition => {
        const section = sections[sectionKey];

        // Collect field paths owned by this section
        const fields = Object.entries(sectionFieldMap)
          .filter(([, sk]) => sk === sectionKey)
          .map(([fieldPath]) => fieldPath);

        // Each generated step renders UIGenerator for its section
        const GeneratedStepComponent = ({
          loadingDefaultsForEdition,
        }: {
          loadingDefaultsForEdition?: boolean;
        }) =>
          React.createElement(UIGenerator, {
            sectionKey,
            sections,
            providerObject,
            loadingDefaultsForEdition,
            namespace,
          });

        return {
          id: getSectionStepId(sectionKey),
          label: section?.label || sectionKey,
          description: section?.description,
          sectionKey,
          component: GeneratedStepComponent,
          fields,
        };
      }),
    [sectionKeys, sections, sectionFieldMap, providerObject, namespace]
  );

  // 3. Merge static + generated steps
  const steps = useMemo(
    () => [...staticSteps, ...generatedSteps],
    [staticSteps, generatedSteps]
  );

  // 4. Build complete field → step-ID map
  const fieldToStepMap = useMemo(() => {
    const map: Record<string, string> = {};

    for (const step of staticSteps) {
      for (const field of step.fields) {
        map[field] = step.id;
      }
    }

    for (const [fieldPath, sectionKey] of Object.entries(sectionFieldMap)) {
      map[fieldPath] = getSectionStepId(sectionKey);
    }

    return map;
  }, [staticSteps, sectionFieldMap]);

  // 5. Default values
  const defaultValues = useMemo(
    () => getDefaultValues(stableUiSchema, selectedTopology),
    [stableUiSchema, selectedTopology]
  );

  // 6. Post-processor
  const postprocess = useCallback(
    (data: Record<string, unknown>) =>
      postprocessSchemaData(data, {
        schema: stableUiSchema,
        selectedTopology,
      }),
    [stableUiSchema, selectedTopology]
  );

  return useMemo(
    () => ({
      steps,
      sections,
      sectionsOrder,
      zodSchema,
      celDependencyGroups,
      fieldToStepMap,
      defaultValues,
      postprocess,
    }),
    [
      steps,
      sections,
      sectionsOrder,
      zodSchema,
      celDependencyGroups,
      fieldToStepMap,
      defaultValues,
      postprocess,
    ]
  );
};
