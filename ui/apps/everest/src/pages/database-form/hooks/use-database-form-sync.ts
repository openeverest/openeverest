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

import { useEffect, useRef } from 'react';
import { UseFormGetValues, UseFormReset } from 'react-hook-form';
import {
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { getDefaultValues } from 'components/ui-generator/utils/default-values';
import { mergeTopologyDefaults } from 'components/ui-generator/utils/default-values/merge-topology-defaults';
import { InstancePreset } from 'shared-types/api.types';
import { usePresetFormSync } from '../preset-selection';
import { DbWizardType } from '../database-form-schema';

interface PresetSource {
  resolvedPreset: InstancePreset | null;
  presetName: string;
  presetSelected: boolean;
  namespace: string | undefined;
}

interface UseDatabaseFormSyncArgs {
  mode: FormMode;
  uiSchema: TopologyUISchemas;
  defaultValues: Record<string, unknown>;
  defaultTopology: string;
  selectedTopology: string | undefined;
  preset: PresetSource;
  reset: UseFormReset<DbWizardType>;
  getValues: UseFormGetValues<DbWizardType>;
}

// Single owner of programmatic form writes for the create/edit wizard. Each
// "source" is a self-contained sync with its own precedence; today that is
// preset population and topology-default merge. New sources (e.g. plugins)
// compose here, so the form never has competing resets scattered across
// components with ad-hoc ordering guards.
export const useDatabaseFormSync = ({
  mode,
  uiSchema,
  defaultValues,
  defaultTopology,
  selectedTopology,
  preset,
  reset,
  getValues,
}: UseDatabaseFormSyncArgs) => {
  // Source 1 — preset: populates the form from the resolved preset and reverts
  // on clear. Owns topology while a preset is selected.
  usePresetFormSync({
    mode,
    uiSchema,
    defaultValues,
    defaultTopology,
    resolvedPreset: preset.resolvedPreset,
    presetName: preset.presetName,
    namespace: preset.namespace,
    reset,
    getValues,
  });

  // Source 2 — topology defaults: on a manual topology switch (no preset),
  // merge the new topology's defaults over the current values. Registered after
  // the preset source so it runs before the page's revalidation effect.
  const prevTopologyTypeRef = useRef<string | undefined>(undefined);
  useEffect(() => {
    const topologyType = selectedTopology;
    if (!topologyType || !uiSchema) return;
    if (prevTopologyTypeRef.current === undefined) {
      prevTopologyTypeRef.current = topologyType;
      return;
    }
    if (topologyType === prevTopologyTypeRef.current) return;
    prevTopologyTypeRef.current = topologyType;

    // Preset owns topology while selected; its own sync handles population.
    if (preset.presetSelected) return;

    const topologyDefaults = getDefaultValues(uiSchema, topologyType);
    const merged = mergeTopologyDefaults(
      getValues() as Record<string, unknown>,
      topologyDefaults
    );
    reset(merged as DbWizardType, { keepDirty: true, keepIsSubmitted: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedTopology, uiSchema]);
};
