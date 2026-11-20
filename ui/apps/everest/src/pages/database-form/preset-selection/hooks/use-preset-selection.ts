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

import { useEffect, useMemo, useState } from 'react';
import { useClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import { useInstancePresets } from 'hooks/api/instance-presets';
import { PresetSelectionContextType } from '../preset-selection-context';
import { DroppedPreset, PresetScope } from '../preset-selection.types';
import { mergePresets, parsePresetId } from '../preset-selection.utils';

// The picked preset ID lives in react-hook-form (DbWizardFormFields.presetId)
// and is passed in; the selected spec is taken from the already-fetched lists.
export const usePresetSelection = (
  clusterName: string,
  provider: string | undefined,
  namespace: string,
  presetId: string
): PresetSelectionContextType => {
  const clusterPresetsQuery = useClusterInstancePresets(clusterName, provider);
  const namespacePresetsQuery = useInstancePresets(
    clusterName,
    namespace,
    provider
  );

  const presets = useMemo(
    () =>
      mergePresets(
        clusterPresetsQuery.data,
        namespacePresetsQuery.data,
        namespace
      ),
    [clusterPresetsQuery.data, namespacePresetsQuery.data, namespace]
  );

  const selectedPreset = useMemo(
    () => presets.find((preset) => preset.id === presetId) ?? null,
    [presets, presetId]
  );

  const pickedId = parsePresetId(presetId);
  const pickedFromOtherNamespace =
    pickedId?.scope === PresetScope.Namespace &&
    pickedId.namespace !== namespace;
  const isOwnerListFetching =
    pickedId?.scope === PresetScope.Namespace
      ? namespacePresetsQuery.isFetching
      : clusterPresetsQuery.isFetching;
  // A missing preset is only "unavailable" once the list it belongs to has
  // settled; until then it may simply not have loaded yet.
  const presetUnavailable =
    Boolean(presetId) &&
    !selectedPreset &&
    (pickedFromOtherNamespace || !isOwnerListFetching);

  const [droppedPreset, setDroppedPreset] = useState<DroppedPreset | null>(
    null
  );

  useEffect(() => {
    if (presetUnavailable) {
      const dropped = parsePresetId(presetId);
      setDroppedPreset(
        dropped
          ? {
              name: dropped.name,
              namespace,
              presetNamespace:
                dropped.scope === PresetScope.Namespace
                  ? dropped.namespace
                  : undefined,
            }
          : null
      );
      return;
    }
    // The notice is about one namespace and ends with the next pick.
    setDroppedPreset((current) =>
      current && (presetId || current.namespace !== namespace) ? null : current
    );
  }, [presetUnavailable, presetId, namespace]);

  return useMemo<PresetSelectionContextType>(
    () => ({
      presets,
      // Cluster presets do not depend on the namespace, so they never wait
      // for (or disappear with) the namespace list.
      isLoadingPresets: clusterPresetsQuery.isLoading,
      isError: clusterPresetsQuery.isError || namespacePresetsQuery.isError,
      presetId,
      selectedPreset,
      presetSelected: Boolean(presetId),
      presetPending: Boolean(presetId) && !selectedPreset,
      presetUnavailable,
      droppedPreset,
    }),
    [
      presets,
      clusterPresetsQuery.isLoading,
      clusterPresetsQuery.isError,
      namespacePresetsQuery.isError,
      presetId,
      selectedPreset,
      presetUnavailable,
      droppedPreset,
    ]
  );
};
