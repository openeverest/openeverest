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

import { CrdsGen, HttpApi } from '@generated/api-types';
import { Section } from 'components/ui-generator/ui-generator.types';

export type Backup = CrdsGen.components['schemas']['Backup'];
export type BackupClass = CrdsGen.components['schemas']['BackupClass'];
export type BackupList = CrdsGen.components['schemas']['BackupList'];
export type BackupClassList = CrdsGen.components['schemas']['BackupClassList'];

export type CreateBackupPayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/backups']['post']['requestBody']['content']['application/json'];
export type CreateBackupResponse =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/backups']['post']['responses']['201']['content']['application/json'];
export type GetBackupPayload =
  HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/backups/{backup}']['get']['responses']['200']['content']['application/json'];
export type DeleteBackupPayload =
  | HttpApi.paths['/clusters/{cluster}/namespaces/{namespace}/backups/{backup}']['delete']['responses']['200']['content']['application/json']
  | void;
export type GetBackupClassPayload =
  HttpApi.paths['/clusters/{cluster}/backup-classes/{backupClass}']['get']['responses']['200']['content']['application/json'];
export type ListBackupClassesPayload =
  HttpApi.paths['/clusters/{cluster}/backup-classes']['get']['responses']['200']['content']['application/json'];

// Raw backup state as described by the generated API types.
// Today this is just `string`, but if the OpenAPI schema becomes enum-constrained
// this alias will narrow automatically.
export type BackupStateFromAPI = NonNullable<
  NonNullable<Backup['status']>['state']
>;

export const BackupStatus = {
  PENDING: 'Pending',
  RUNNING: 'Running',
  SUCCEEDED: 'Succeeded',
  FAILED: 'Failed',
  ERROR: 'Error',
  DELETING: 'Deleting',
  UNKNOWN: 'Unknown',
} as const;

// UI-normalized backup state. `UNKNOWN` is owned by the UI as a fallback for
// missing or unmapped API values.
export type NormalizedBackupStatus =
  | BackupStateFromAPI
  | (typeof BackupStatus)['UNKNOWN'];

/** Normalize a Backup's state to a safe UI value. Centralizes the `?? UNKNOWN` fallback. */
export const getBackupState = (backup: Backup): NormalizedBackupStatus =>
  (backup.status?.state as NormalizedBackupStatus) ?? BackupStatus.UNKNOWN;

// The generated API types define uiSchema as `Record<string, never>` (opaque),
// so we maintain this typed alias for UIGenerator consumption.
export type BackupClassUiSchemaSections = {
  backup?: Section;
  pitr?: Section;
  restore?: Section;
};

export type DatabaseClusterPitrPayload =
  | {
      earliestDate: string;
      latestDate: string;
      latestBackupName: string;
      gaps: boolean;
    }
  | Record<string, never>;

export type DatabaseClusterPitr = {
  earliestDate: Date;
  latestDate: Date;
  latestBackupName: string;
  gaps: boolean;
};
