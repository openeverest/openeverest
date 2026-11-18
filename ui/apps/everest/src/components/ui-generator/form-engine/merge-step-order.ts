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

import {
  BASE_STEP_ID,
  RESERVED_STATIC_STEP_IDS,
} from 'pages/database-form/database-form-body/steps/constants';

export type OrderableStep = {
  id: string;
  sectionKey?: string;
};

export const mergeStepOrder = <T extends OrderableStep>(
  staticSteps: T[],
  generatedSteps: T[],
  sectionsOrder?: readonly string[]
): T[] => {
  const baseStep = staticSteps.find((step) => step.id === BASE_STEP_ID);
  const staticById = new Map(staticSteps.map((step) => [step.id, step]));
  const reservedStaticIds = new Set<string>(RESERVED_STATIC_STEP_IDS);
  const explicitlyListedStaticIds = new Set<string>();

  for (const key of sectionsOrder ?? []) {
    if (reservedStaticIds.has(key) && staticById.has(key)) {
      explicitlyListedStaticIds.add(key);
    }
  }

  const ordered: T[] = [];
  const emitted = new Set<string>();

  const emit = (step: T) => {
    if (!emitted.has(step.id)) {
      emitted.add(step.id);
      ordered.push(step);
    }
  };

  if (baseStep) {
    emit(baseStep);
  }

  for (const step of staticSteps) {
    if (step.id !== BASE_STEP_ID && !explicitlyListedStaticIds.has(step.id)) {
      emit(step);
    }
  }

  const emittedGeneratedKeys = new Set<string>();

  for (const key of sectionsOrder ?? []) {
    if (key === BASE_STEP_ID) continue;

    if (reservedStaticIds.has(key)) {
      const staticStep = staticById.get(key);
      if (staticStep) {
        emit(staticStep);
        emittedGeneratedKeys.add(key);
        continue;
      }
    }

    const generatedStep = generatedSteps.find(
      (step) => step.sectionKey === key
    );
    if (generatedStep) {
      emit(generatedStep);
      emittedGeneratedKeys.add(key);
    }
  }

  for (const step of generatedSteps) {
    const collidesWithStaticStep =
      step.sectionKey &&
      reservedStaticIds.has(step.sectionKey) &&
      staticById.has(step.sectionKey);

    if (
      !collidesWithStaticStep &&
      (!step.sectionKey || !emittedGeneratedKeys.has(step.sectionKey))
    ) {
      emit(step);
    }
  }

  return ordered;
};
