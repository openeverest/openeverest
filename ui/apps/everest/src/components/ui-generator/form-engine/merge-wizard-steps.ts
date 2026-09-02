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

import { StepDefinition } from './types';

export const DEFAULT_PINNED_STEP_ID = 'base';
export const DEFAULT_RESERVED_STATIC_STEP_IDS: ReadonlySet<string> = new Set([
  'base',
  'import',
  'backups',
]);

export interface MergeWizardStepsOptions {
  pinnedStepId?: string;
  reservedStaticStepIds?: ReadonlySet<string>;
}

export const mergeWizardSteps = (
  staticSteps: StepDefinition[],
  generatedSteps: Map<string, StepDefinition>,
  sectionsOrder?: string[],
  options?: MergeWizardStepsOptions
): StepDefinition[] => {
  const pinnedStepId = options?.pinnedStepId ?? DEFAULT_PINNED_STEP_ID;
  const reservedStaticStepIds =
    options?.reservedStaticStepIds ?? DEFAULT_RESERVED_STATIC_STEP_IDS;

  const baseStep = staticSteps.find((s) => s.id === pinnedStepId);
  const nonBaseStaticSteps = staticSteps.filter((s) => s.id !== pinnedStepId);
  const activeStaticStepMap = new Map(
    nonBaseStaticSteps.map((step) => [step.id, step])
  );

  // Step 0: Filter collisions across all reserved static step IDs.
  // Schema sections sharing a key with any reserved static step ID are dropped.
  const collisionIds = new Set([...reservedStaticStepIds, pinnedStepId]);
  const filteredGeneratedSteps = new Map<string, StepDefinition>();
  for (const [key, step] of generatedSteps.entries()) {
    if (!collisionIds.has(key)) {
      filteredGeneratedSteps.set(key, step);
    }
  }

  const result: StepDefinition[] = [];
  if (baseStep) {
    result.push(baseStep);
  }

  // Step 2: When no explicit sectionsOrder is provided, keep default ordering
  if (!sectionsOrder || sectionsOrder.length === 0) {
    for (const staticStep of nonBaseStaticSteps) {
      result.push(staticStep);
    }
    for (const genStep of filteredGeneratedSteps.values()) {
      result.push(genStep);
    }
    return result;
  }

  // Step 3: Partition non-base static steps into listed vs unlisted
  const orderSet = new Set(sectionsOrder);
  const unlistedStaticSteps = nonBaseStaticSteps.filter(
    (step) => !orderSet.has(step.id)
  );

  // Unlisted static steps retain their default position before generated sections
  for (const staticStep of unlistedStaticSteps) {
    result.push(staticStep);
  }

  // Step 4: Walk sectionsOrder
  const addedStepIds = new Set(result.map((step) => step.id));

  for (const key of sectionsOrder) {
    if (key === pinnedStepId) {
      continue;
    }

    const staticStep = activeStaticStepMap.get(key);
    if (staticStep && !addedStepIds.has(staticStep.id)) {
      result.push(staticStep);
      addedStepIds.add(staticStep.id);
      continue;
    }

    const genStep = filteredGeneratedSteps.get(key);
    if (genStep && !addedStepIds.has(genStep.id)) {
      result.push(genStep);
      addedStepIds.add(genStep.id);
    }
  }

  // Step 5: Append any remaining generated steps not referenced in sectionsOrder
  for (const genStep of filteredGeneratedSteps.values()) {
    if (!addedStepIds.has(genStep.id)) {
      result.push(genStep);
      addedStepIds.add(genStep.id);
    }
  }

  return result;
};
