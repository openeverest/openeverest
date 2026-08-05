// everest
// Copyright (C) 2023 Percona LLC
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

import { HttpApi } from '@generated/api-types';

export type GetRestorePayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/instances/{instance}/restores']['get']['responses']['200']['content']['application/json'];

export type CreateRestorePayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/restores']['post']['requestBody']['content']['application/json'];

export type RestoreDataSource = CreateRestorePayload['spec']['dataSource'];

// Discriminated FE input describing a restore intent. An in-place restore leaves
// the source instance implicit; a clone into a new DB names it explicitly via
// sourceInstanceName, since the new instance has no PITR stream of its own.
// storageName on the Backup arm is an FE-only hint (the seeding storage of the
// backup) used to register that storage on a clone — it is NOT part of the
// payload, since the BE resolves the storage from the referenced Backup.
export type RestoreDataSourceInput =
  | { type: 'Backup'; backupName: string; storageName?: string }
  | {
      type: 'PointInTime';
      storageName: string;
      recoveryTarget: 'latest';
      sourceInstanceName?: string;
    }
  | {
      type: 'PointInTime';
      storageName: string;
      recoveryTarget: 'date';
      date: string;
      sourceInstanceName?: string;
    };

export type RestoreCreateVariables = {
  instanceName: string;
} & RestoreDataSourceInput;

export type Restore = {
  name: string;
  startTime: string;
  endTime?: string;
  type: 'full' | 'pitr' | 'import';
  state: string;
  backupSource: string;
};

export enum PXC_STATUS {
  STARTING = 'Starting',
  STOPPING = 'Stopping Cluster',
  RESTORING = 'Restoring',
  STARTING_CLUSTER = 'Starting Cluster',
  PREPARING_CLUSTER = 'Preparing Cluster',
  PITR_RECOVERING = 'Point-in-time recovering',
  FAILED = 'Failed',
  SUCCEEDED = 'Succeeded',
}

export enum PSMDB_STATUS {
  WAITING = 'waiting',
  REQUESTED = 'requested',
  REJECTED = 'rejected',
  RUNNING = 'running',
  ERROR = 'error',
  READY = 'ready',
}

export enum PG_STATUS {
  STARTING = 'Starting',
  RUNNING = 'Running',
  FAILED = 'Failed',
  SUCCEEDED = 'Succeeded',
}
