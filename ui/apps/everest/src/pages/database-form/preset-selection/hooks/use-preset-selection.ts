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

import { useMemo } from 'react';
import { useClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import { PresetSelectionContextType } from '../preset-selection-context';

// The picked preset NAME lives in react-hook-form (PRESET_NAME_FIELD) and is
// passed in; the selected spec is taken from the already-fetched preset list.
export const usePresetSelection = (
  clusterName: string,
  provider: string | undefined,
  presetName: string
): PresetSelectionContextType => {
  const {
    data: presets = [],
    isLoading: isLoadingPresets,
    isError,
  } = useClusterInstancePresets(clusterName, provider);

  const selectedPreset = useMemo(
    () =>
      presets.find((preset) => preset.metadata?.name === presetName) ?? null,
    [presets, presetName]
  );

  return useMemo<PresetSelectionContextType>(
    () => ({
      presets,
      isLoadingPresets,
      isError,
      presetName,
      selectedPreset,
      presetSelected: Boolean(presetName),
      presetPending: Boolean(presetName) && !selectedPreset,
    }),
    [presets, isLoadingPresets, isError, presetName, selectedPreset]
  );
};
