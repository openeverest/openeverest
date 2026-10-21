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

export const Messages = {
  pageTitle: 'Import database',
  pageDescription:
    'Choose a backup to create a new database from. You will configure the database in the next step.',
  labels: {
    namespace: 'Namespace',
    backupClass: 'Backup class',
    provider: 'Database provider',
  },
  validation: {
    namespace: 'Select a namespace.',
    storage: 'Select a backup storage.',
    backupClass: 'Select a backup class.',
    provider: 'Select a database provider.',
    backup: 'Select a backup.',
  },
  noNamespaces: "You don't have permission to import backups in any namespace.",
  noImportClasses: 'No backup class supports importing backups.',
  noStorages:
    'To import a database, add the backup storage that holds its backups.',
  addStorage: 'Add storage',
  providerNotInstalled:
    'The database provider of this backup class is not installed.',
  backups: {
    scan: 'Scan storage',
    scanAgain: 'Scan again',
    scanning: 'Scanning…',
    completedAt: 'Completed',
    path: 'Path',
    version: 'Version',
    type: 'Type',
    topology: 'Topology',
    noBackups: 'No backups found yet. Scan the storage to find them.',
  },
  scan: {
    running: 'Scanning the storage for backups. This can take a minute.',
    stalled:
      'The scan is taking longer than expected. The backup provider may be unavailable. You can scan again.',
    retrying: (message: string) =>
      `The scan hit an error and is retrying: ${message}`,
    failed: (date: string, message: string) =>
      `The last scan, on ${date}, failed: ${message}`,
    succeeded: (date: string, count: number) =>
      `The last scan, on ${date}, found ${count} ${count === 1 ? 'backup' : 'backups'}.`,
  },
};
