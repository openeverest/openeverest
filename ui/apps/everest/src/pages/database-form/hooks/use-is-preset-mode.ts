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

import { useFormContext } from 'react-hook-form';
import { DbWizardFormFields } from 'consts';

/**
 * Hook to determine if form should be read-only based on preset selection.
 * Phase 1: Form is read-only when a preset is selected.
 * Phase 2: Form will be editable with preset overrides.
 */
export const useIsPresetMode = (): boolean => {
  const { watch } = useFormContext();
  const selectedPreset = watch(DbWizardFormFields.instancePreset);

  // Check if preset is selected (handle both object and string formats)
  const hasPreset = selectedPreset
    ? typeof selectedPreset === 'object'
      ? !!selectedPreset.value
      : !!selectedPreset
    : false;

  // Phase 1: Return true when preset is selected (form will be read-only)
  // Phase 2: This will be updated to allow editing
  return hasPreset;
};
