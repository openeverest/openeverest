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
import {
  useInstancePresets,
  useResolveInstancePreset,
} from 'hooks/api/instance-presets';
import { PresetSelectionContextType } from './preset-selection.types';
import { Messages } from './preset-select.messages';

// Resolves the picked preset for the current namespace and exposes the list +
// gating flags. The picked preset NAME lives in react-hook-form (PRESET_NAME_FIELD)
// and is passed in; this hook only owns the async resolved spec.
export const usePresetSelection = (
  clusterName: string,
  provider: string | undefined,
  namespace: string | undefined,
  presetName: string
): PresetSelectionContextType => {
  const {
    data: presets = [],
    isLoading: isLoadingPresets,
    isError,
  } = useInstancePresets(clusterName, provider);

  // Re-resolves automatically when presetName or namespace changes (query key);
  // react-query dedupes/cancels stale fetches.
  const {
    data: resolved,
    isFetching: isResolving,
    isError: hasResolveError,
  } = useResolveInstancePreset(clusterName, presetName, namespace);

  const resolvedPreset = resolved ?? null;
  const resolveError = hasResolveError ? Messages.resolveError : null;

  return useMemo<PresetSelectionContextType>(
    () => ({
      presets,
      isLoadingPresets,
      isError,
      presetName,
      resolvedPreset,
      isResolving,
      resolveError,
      presetSelected: Boolean(presetName),
      presetPending: Boolean(presetName) && (!resolvedPreset || isResolving),
    }),
    [
      presets,
      isLoadingPresets,
      isError,
      presetName,
      resolvedPreset,
      isResolving,
      resolveError,
    ]
  );
};
