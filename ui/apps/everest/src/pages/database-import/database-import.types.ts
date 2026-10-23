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

import { AlertColor } from '@mui/material';
import { Backup, BackupImportState } from 'shared-types/backups.types';

type ExternalBackupOrigin = NonNullable<Backup['spec']['origin']['external']>;

export type ImportBackupRow = { name: string } & Pick<
  ExternalBackupOrigin,
  'path' | 'completedAt'
>;

export interface ScanTarget {
  storageName: string;
  backupClassName: string;
}

export interface ScanProgress {
  isScanning: boolean;
  isStalled: boolean;
  state?: BackupImportState;
  message?: string;
  discoveredCount?: number;
  startedAt?: string;
}

export interface ScanAlert {
  severity: AlertColor;
  text: string;
}
