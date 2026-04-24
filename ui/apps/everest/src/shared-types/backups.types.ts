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

export type SingleBackupPayload = {
  metadata: {
    name: string;
  };
  status?: {
    created: string;
    completed: string;
    state: string;
    size?: string;
  };
  spec: {
    dbClusterName: string;
    backupStorageName: string;
  };
};

export type GetBackupsPayload = {
  items: Array<SingleBackupPayload>;
};

export type Backup = {
  name: string;
  created?: string;
  completed?: string;
  size?: string;
  state: BackupStatus;
  dbClusterName: string;
  backupStorageName: string;
};

export enum BackupStatus {
  OK = 'Succeeded',
  FAILED = 'Failed',
  IN_PROGRESS = 'In progress',
  UNKNOWN = 'Unknown',
  DELETING = 'Deleting',
}

export type BackupPayload = {
  apiVersion: 'everest.percona.com/v1alpha1';
  kind: 'DatabaseClusterBackup';
  metadata: {
    name: string;
  };
  spec: {
    dbClusterName: string;
    backupStorageName: string;
  };
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
