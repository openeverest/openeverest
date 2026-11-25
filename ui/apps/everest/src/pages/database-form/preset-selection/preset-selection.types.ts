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

import { ClusterInstancePreset, InstancePreset } from 'shared-types/api.types';

export enum PresetScope {
  Cluster = 'cluster',
  Namespace = 'namespace',
}

// A preset of either kind as the wizard offers it. A ClusterInstancePreset and
// an InstancePreset may share a name, so `id` (not `name`) identifies the pick.
export interface SelectablePreset {
  id: string;
  scope: PresetScope;
  name: string;
  spec: ClusterInstancePreset['spec'] | InstancePreset['spec'];
}

export type PresetIdParts =
  | { scope: PresetScope.Cluster; name: string }
  | { scope: PresetScope.Namespace; namespace: string; name: string };

// Where the picked preset stands: nothing picked, its list still loading, its
// spec at hand, or gone (it belongs to another namespace, or was removed).
export type PresetStatus = 'none' | 'loading' | 'ready' | 'unavailable';

// What the two preset lists and the pick add up to.
export interface PresetSelection {
  presets: SelectablePreset[];
  isLoadingPresets: boolean;
  // True when either list failed; the other one is still offered.
  isError: boolean;
  // Requests again only the lists that failed.
  retryFailedLists: () => void;
  isRetrying: boolean;
  presetId: string;
  selectedPreset: SelectablePreset | null;
  status: PresetStatus;
}

// A picked preset the wizard had to drop, kept to tell the user why.
export interface DroppedPreset {
  name: string;
  wizardNamespace: string;
  ownerNamespace?: string;
}

export interface DroppedPresetNotice {
  droppedPreset: DroppedPreset | null;
  dismissDroppedPreset: () => void;
}
