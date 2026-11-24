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
import { DbWizardType } from '../../database-form-schema';
import { SelectablePreset } from '../preset-selection.types';
import { getPresetBackupFormValues } from '../preset-selection.utils';

interface UsePresetFormSyncArgs {
  mode: FormMode;
  uiSchema: TopologyUISchemas;
  defaultValues: Record<string, unknown>;
  defaultTopology: string;
  selectedPreset: SelectablePreset | null;
  presetId: string;
  presetUnavailable: boolean;
  reset: UseFormReset<DbWizardType>;
  getValues: UseFormGetValues<DbWizardType>;
}

// Populates the wizard form from the picked preset (so preview/fields reflect
// it) and reverts to defaults when the preset is cleared or can no longer be
// resolved. New mode only, and idempotent per preset so it never clobbers the
// user's dbName / namespace or edits made outside preset mode.
export const usePresetFormSync = ({
  mode,
  uiSchema,
  defaultValues,
  defaultTopology,
  selectedPreset,
  presetId,
  presetUnavailable,
  reset,
  getValues,
}: UsePresetFormSyncArgs) => {
  const appliedRef = useRef<string | null>(null);

  useEffect(() => {
    if (mode !== FormMode.New) return;

    // Preset cleared or gone → revert once to the pristine defaults. Dropping
    // an unavailable pick here keeps the create from staying blocked on a
    // spec that will never arrive.
    if (!presetId || presetUnavailable) {
      const {
        dbName,
        k8sNamespace,
        provider,
        presetId: presetIdInForm,
      } = getValues();
      // Checked against the form, not the prop, so the revert runs once.
      if (appliedRef.current || presetIdInForm) {
        reset({
          ...defaultValues,
          provider,
          dbName,
          k8sNamespace,
          presetId: '',
        } as DbWizardType);
        appliedRef.current = null;
      }
      return;
    }

    // Wait until the picked preset is present in the loaded list.
    if (!selectedPreset) return;

    if (appliedRef.current === presetId) return;

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
      // Backups and monitoring are not ui-generator sections, so they are
      // mapped onto the wizard's own form shape for the preview to show what
      // the preset will actually deploy.
      backup: getPresetBackupFormValues(selectedPreset.spec),
      monitoring: {
        destinations: selectedPreset.spec?.monitoring?.destinations ?? [],
      },
      topology: { type: presetTopology },
      provider,
      dbName,
      k8sNamespace,
      presetId,
    } as unknown as DbWizardType);

    appliedRef.current = presetId;
  }, [
    mode,
    uiSchema,
    defaultValues,
    defaultTopology,
    selectedPreset,
    presetId,
    presetUnavailable,
    reset,
    getValues,
  ]);
};
