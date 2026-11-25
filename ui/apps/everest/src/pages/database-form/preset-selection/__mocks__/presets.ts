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
import { PresetScope, SelectablePreset } from '../preset-selection.types';
import {
  getClusterPresetId,
  getNamespacePresetId,
} from '../preset-selection.utils';

export const PRESET_PROVIDER = 'percona-server-mongodb';
// The cluster preset and both namespace presets share one name on purpose:
// the wizard has to tell them apart by scope and namespace.
export const PRESET_NAME = 'psmdb-default';
export const TEAM_A = 'team-a';
export const TEAM_B = 'team-b';

export const clusterPresetSpec: SelectablePreset['spec'] = {
  providerRef: { name: PRESET_PROVIDER },
  version: '8.0.12',
  topology: { type: 'replicaSet' },
  components: {
    engine: {
      replicas: 3,
      resources: { limits: { cpu: 1, memory: '4Gi' } },
      storage: { size: '25Gi' },
    },
  },
};

// What a namespace preset is for: references that only exist in its namespace.
export const teamAPresetSpec: SelectablePreset['spec'] = {
  ...clusterPresetSpec,
  version: '7.0.18',
  backup: {
    enabled: true,
    classRef: { name: 'psmdb-backup-class' },
    storages: [
      {
        storageRef: { name: 'bs-team-a' },
        schedules: [{ name: 'daily', cron: '0 0 * * *', enabled: true }],
        pitr: { enabled: true },
      },
    ],
  },
  monitoring: {
    destinations: [
      { name: 'pmm-team-a', destinationRef: { name: 'pmm-team-a' } },
    ],
  },
};

export const teamBPresetSpec: SelectablePreset['spec'] = {
  ...clusterPresetSpec,
  version: '6.0.21',
};

// As the API lists them.
export const clusterPresetItem: ClusterInstancePreset = {
  metadata: { name: PRESET_NAME },
  spec: clusterPresetSpec,
};

export const teamAPresetItem: InstancePreset = {
  metadata: { name: PRESET_NAME, namespace: TEAM_A },
  spec: teamAPresetSpec,
};

export const teamBPresetItem: InstancePreset = {
  metadata: { name: PRESET_NAME, namespace: TEAM_B },
  spec: teamBPresetSpec,
};

// As the wizard offers them.
export const clusterPreset: SelectablePreset = {
  id: getClusterPresetId(PRESET_NAME),
  scope: PresetScope.Cluster,
  name: PRESET_NAME,
  spec: clusterPresetSpec,
};

export const teamAPreset: SelectablePreset = {
  id: getNamespacePresetId(TEAM_A, PRESET_NAME),
  scope: PresetScope.Namespace,
  name: PRESET_NAME,
  spec: teamAPresetSpec,
};
