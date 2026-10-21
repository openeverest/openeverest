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

import { format } from 'date-fns';
import { DATE_FORMAT } from 'consts';
import { ImportNavigation } from 'pages/database-form/import-navigation-state';
import { Provider } from 'shared-types/api.types';
import {
  Backup,
  BackupClass,
  BackupImport,
  BackupImportState,
} from 'shared-types/backups.types';
import { backupClassSupportsProvider } from 'utils/backup-classes';
import { DATABASE_IMPORT_DEFAULT_VALUES } from './database-import.constants';
import { Messages } from './database-import.messages';
import {
  DatabaseImportFields,
  DatabaseImportFormValues,
} from './database-import-schema';
import {
  ImportBackupRow,
  ScanAlert,
  ScanProgress,
  ScanTarget,
} from './database-import.types';

// Error is transient: the controller keeps retrying it.
const FINISHED_SCAN_STATES: BackupImportState[] = ['Succeeded', 'Failed'];

// Coming back from the wizard ("Change backup") restores the previous choice.
export const getImportFormDefaults = (
  navigation: ImportNavigation | undefined
): DatabaseImportFormValues =>
  navigation
    ? {
        ...DATABASE_IMPORT_DEFAULT_VALUES,
        [DatabaseImportFields.namespace]: navigation.importSource.namespace,
        [DatabaseImportFields.backupClassName]:
          navigation.importSource.backupClassName,
        [DatabaseImportFields.providerName]: navigation.providerName,
        [DatabaseImportFields.backupName]:
          navigation.importSource.dataSource.backupName,
      }
    : DATABASE_IMPORT_DEFAULT_VALUES;

export const getImportBackupRows = (backups: Backup[]): ImportBackupRow[] =>
  backups
    .flatMap((backup) => {
      const name = backup.metadata?.name;
      const external = backup.spec.origin.external;
      if (!name || !external || backup.metadata?.deletionTimestamp) {
        return [];
      }
      return [{ name, path: external.path, completedAt: external.completedAt }];
    })
    .sort((a, b) => Date.parse(b.completedAt) - Date.parse(a.completedAt));

export const getClassProviders = (
  backupClass: BackupClass | undefined,
  providers: Provider[]
): Provider[] =>
  backupClass
    ? providers.filter((p) =>
        backupClassSupportsProvider(backupClass, p.metadata?.name ?? '')
      )
    : [];

export const isBackupImportFinished = (backupImport: BackupImport) => {
  const state = backupImport.status?.state;
  return !!state && FINISHED_SCAN_STATES.includes(state);
};

// The newest scan of the storage with the class, whoever started it.
export const findLatestBackupImport = (
  imports: BackupImport[],
  { storageName, backupClassName }: ScanTarget
): BackupImport | undefined =>
  imports
    .filter(
      ({ metadata, spec }) =>
        spec.storageRef.name === storageName &&
        spec.classRef.name === backupClassName &&
        !metadata?.deletionTimestamp
    )
    .reduce<
      BackupImport | undefined
    >((latest, current) => (!latest || Date.parse(current.metadata?.creationTimestamp ?? '') > Date.parse(latest.metadata?.creationTimestamp ?? '') ? current : latest), undefined);

export const getScanAlert = ({
  isScanning,
  isStalled,
  state,
  message = '',
  discoveredCount = 0,
  startedAt,
}: ScanProgress): ScanAlert | undefined => {
  if (state === 'Error') {
    return { severity: 'warning', text: Messages.scan.retrying(message) };
  }
  if (isScanning) {
    return isStalled
      ? { severity: 'warning', text: Messages.scan.stalled }
      : undefined;
  }
  const date = startedAt ? format(new Date(startedAt), DATE_FORMAT) : '';
  if (state === 'Failed') {
    return { severity: 'error', text: Messages.scan.failed(date, message) };
  }
  if (state === 'Succeeded') {
    return {
      severity: 'success',
      text: Messages.scan.succeeded(date, discoveredCount),
    };
  }
  return undefined;
};
