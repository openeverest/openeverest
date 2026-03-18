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

import { useCallback, useEffect, useMemo, useState } from 'react';
import { StepDefinition } from './types';

export type StepNavigationResult = {
  /** ID of the currently active step. */
  activeStepId: string;
  /** Zero-based index of the active step. */
  activeStepIndex: number;
  totalSteps: number;
  isFirstStep: boolean;
  isLastStep: boolean;
  stepIdToIndex: Record<string, number>;
  next: () => void;
  back: () => void;
  goTo: (stepId: string) => void;
};

/**
 * Manages wizard step navigation.
 *
 * Pure state machine — no form / validation awareness.  The caller
 * decides when to allow / block transitions.
 */
export const useStepNavigation = (
  steps: StepDefinition[],
  initialStepId?: string
): StepNavigationResult => {
  const fallbackId = steps[0]?.id ?? '';
  const [activeStepId, setActiveStepId] = useState(initialStepId ?? fallbackId);

  const orderedIds = useMemo(() => steps.map((s) => s.id), [steps]);

  const stepIdToIndex = useMemo(
    () =>
      Object.fromEntries(orderedIds.map((id, idx) => [id, idx])) as Record<
        string,
        number
      >,
    [orderedIds]
  );

  const activeStepIndex = stepIdToIndex[activeStepId] ?? 0;
  const totalSteps = steps.length;

  // If the current step disappears (e.g. topology change), reset to first.
  useEffect(() => {
    if (!(activeStepId in stepIdToIndex)) {
      setActiveStepId(fallbackId);
    }
  }, [activeStepId, stepIdToIndex, fallbackId]);

  const next = useCallback(() => {
    if (activeStepIndex < totalSteps - 1) {
      setActiveStepId(orderedIds[activeStepIndex + 1]);
    }
  }, [activeStepIndex, totalSteps, orderedIds]);

  const back = useCallback(() => {
    if (activeStepIndex > 0) {
      setActiveStepId(orderedIds[activeStepIndex - 1]);
    }
  }, [activeStepIndex, orderedIds]);

  const goTo = useCallback(
    (stepId: string) => {
      if (stepId in stepIdToIndex) {
        setActiveStepId(stepId);
      }
    },
    [stepIdToIndex]
  );

  return useMemo(
    () => ({
      activeStepId,
      activeStepIndex,
      totalSteps,
      isFirstStep: activeStepIndex === 0,
      isLastStep: activeStepIndex === totalSteps - 1,
      stepIdToIndex,
      next,
      back,
      goTo,
    }),
    [activeStepId, activeStepIndex, totalSteps, stepIdToIndex, next, back, goTo]
  );
};
