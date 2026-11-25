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

import { useCallback, useMemo, useState } from 'react';
import { useClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import { useInstancePresets } from 'hooks/api/instance-presets';
import {
  PresetScope,
  PresetSelection,
  PresetStatus,
} from '../preset-selection.types';
import { mergePresets, parsePresetId } from '../preset-selection.utils';

// The picked preset ID lives in react-hook-form (DbWizardFormFields.presetId)
// and is passed in; the selected spec is taken from the already-fetched lists.
// This only derives where the pick stands: acting on it (filling the form,
// dropping a pick that no longer applies) is usePresetFormSync's job.
export const usePresetSelection = (
  clusterName: string,
  provider: string | undefined,
  namespace: string,
  presetId: string
): PresetSelection => {
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

  // The cards first appear once both lists have arrived, so the form is not
  // pushed down twice. After that a namespace switch reloads only the
  // namespace presets, and the cluster ones must stay on screen meanwhile.
  const isLoadingAnyList =
    clusterPresetsQuery.isLoading || namespacePresetsQuery.isLoading;
  const [hasShownPresets, setHasShownPresets] = useState(false);
  if (!hasShownPresets && !isLoadingAnyList) {
    setHasShownPresets(true);
  }
  const isLoadingPresets = hasShownPresets
    ? clusterPresetsQuery.isLoading
    : isLoadingAnyList;

  const pickedId = parsePresetId(presetId);
  const pickedFromOtherNamespace =
    pickedId?.scope === PresetScope.Namespace &&
    pickedId.namespace !== namespace;
  const isOwnerListFetching =
    pickedId?.scope === PresetScope.Namespace
      ? namespacePresetsQuery.isFetching
      : clusterPresetsQuery.isFetching;

  let status: PresetStatus = 'none';
  if (selectedPreset) {
    status = 'ready';
  } else if (presetId) {
    // A missing preset is gone only once the list it belongs to has settled;
    // until then it may simply not have loaded yet.
    status =
      pickedFromOtherNamespace || !isOwnerListFetching
        ? 'unavailable'
        : 'loading';
  }

  const clusterListFailed = clusterPresetsQuery.isError;
  const namespaceListFailed = namespacePresetsQuery.isError;
  const refetchClusterList = clusterPresetsQuery.refetch;
  const refetchNamespaceList = namespacePresetsQuery.refetch;
  const retryFailedLists = useCallback(() => {
    if (clusterListFailed) {
      refetchClusterList();
    }
    if (namespaceListFailed) {
      refetchNamespaceList();
    }
  }, [
    clusterListFailed,
    namespaceListFailed,
    refetchClusterList,
    refetchNamespaceList,
  ]);
  const isRetrying =
    (clusterListFailed && clusterPresetsQuery.isFetching) ||
    (namespaceListFailed && namespacePresetsQuery.isFetching);

  return useMemo<PresetSelection>(
    () => ({
      presets,
      isLoadingPresets,
      isError: clusterListFailed || namespaceListFailed,
      retryFailedLists,
      isRetrying,
      presetId,
      selectedPreset,
      status,
    }),
    [
      presets,
      isLoadingPresets,
      clusterListFailed,
      namespaceListFailed,
      retryFailedLists,
      isRetrying,
      presetId,
      selectedPreset,
      status,
    ]
  );
};
