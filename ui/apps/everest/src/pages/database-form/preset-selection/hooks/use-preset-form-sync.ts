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
import { extractInstanceValues } from 'components/ui-generator/utils/default-values/extract-instance-values';
import { ClusterInstancePreset } from 'shared-types/api.types';
import { DbWizardType } from '../../database-form-schema';

interface UsePresetFormSyncArgs {
  mode: FormMode;
  uiSchema: TopologyUISchemas;
  defaultValues: Record<string, unknown>;
  defaultTopology: string;
  selectedPreset: ClusterInstancePreset | null;
  presetName: string;
  reset: UseFormReset<DbWizardType>;
  getValues: UseFormGetValues<DbWizardType>;
}

// Populates the wizard form from the picked preset (so preview/fields reflect
// it) and reverts to defaults when the preset is cleared. New mode only, and
// idempotent per preset so it never clobbers the user's dbName /
// namespace or edits made outside preset mode.
export const usePresetFormSync = ({
  mode,
  uiSchema,
  defaultValues,
  defaultTopology,
  selectedPreset,
  presetName,
  reset,
  getValues,
}: UsePresetFormSyncArgs) => {
  const appliedRef = useRef<string | null>(null);

  useEffect(() => {
    if (mode !== FormMode.New) return;

    // Preset cleared → revert once to the pristine defaults.
    if (!presetName) {
      if (appliedRef.current) {
        const { dbName, k8sNamespace, provider } = getValues();
        reset({
          ...defaultValues,
          provider,
          dbName,
          k8sNamespace,
          presetName: '',
        } as DbWizardType);
        appliedRef.current = null;
      }
      return;
    }

    // Wait until the picked preset is present in the loaded list.
    if (!selectedPreset) return;

    if (appliedRef.current === presetName) return;

    const presetTopology =
      selectedPreset.spec?.topology?.type ?? defaultTopology;
    const sections = uiSchema[presetTopology]?.sections ?? {};
    // Build a complete, valid spec: extractInstanceValues in New mode reads the
    // preset value where present and falls back to the schema/type default
    // otherwise, so preset values win while fields the preset omits keep a valid
    // empty scaffold (e.g. monitoring off) instead of becoming undefined.
    const baseDefaults = getDefaultValues(uiSchema, presetTopology);
    const specValues = extractInstanceValues(
      sections,
      { spec: selectedPreset.spec },
      mode
    );
    const { dbName, k8sNamespace, provider } = getValues();

    reset({
      ...baseDefaults,
      ...specValues,
      backup: defaultValues.backup,
      topology: { type: presetTopology },
      provider,
      dbName,
      k8sNamespace,
      presetName,
    } as unknown as DbWizardType);

    appliedRef.current = presetName;
  }, [
    mode,
    uiSchema,
    defaultValues,
    defaultTopology,
    selectedPreset,
    presetName,
    reset,
    getValues,
  ]);
};
