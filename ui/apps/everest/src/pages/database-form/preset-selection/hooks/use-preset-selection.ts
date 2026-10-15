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
import { useInstancePresets } from 'hooks/api/instance-presets';
import { PresetSelectionContextType } from '../preset-selection-context';
import { Messages } from '../preset-selection.messages';

// Exposes the namespace-resolved preset list + gating flags. The picked preset
// NAME lives in react-hook-form (PRESET_NAME_FIELD) and is passed in; list items
// are already resolved, so the picked one is read straight from the list.
export const usePresetSelection = (
  clusterName: string,
  provider: string | undefined,
  namespace: string | undefined,
  presetName: string
): PresetSelectionContextType => {
  const {
    data: presets = [],
    isLoading: isLoadingPresets,
    isFetching,
    isError,
  } = useInstancePresets(clusterName, namespace, provider);

  const resolvedPreset = useMemo(
    () =>
      presetName
        ? (presets.find((preset) => preset.metadata?.name === presetName) ??
          null)
        : null,
    [presets, presetName]
  );
  const resolveError =
    presetName && isError ? Messages.resolveError : null;

  return useMemo<PresetSelectionContextType>(
    () => ({
      presets,
      isLoadingPresets,
      isError,
      presetName,
      resolvedPreset,
      resolveError,
      presetSelected: Boolean(presetName),
      presetPending: Boolean(presetName) && (!resolvedPreset || isFetching),
    }),
    [
      presets,
      isLoadingPresets,
      isError,
      presetName,
      resolvedPreset,
      isFetching,
      resolveError,
    ]
  );
};
