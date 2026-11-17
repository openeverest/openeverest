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
  add: 'Add storage',
  noData: 'No backup storage set up yet.',
  columns: {
    name: 'Name',
    namespace: 'Namespace',
    type: 'Type',
    location: 'Location',
    status: 'Status',
  },
  details: {
    authMode: 'Authentication',
    credentials: 'Credentials Secret',
    tls: 'TLS',
    tlsSkipVerify: 'Certificate verification skipped',
    tlsCustomCa: (secret: string, key: string) =>
      `CA certificates in Secret ${secret}, key ${key}`,
    tlsSystem: 'System trust store',
    none: 'None',
  },
  edit: 'Edit',
  delete: 'Delete',
  deleteHeader: 'Delete backup storage',
  deleteConfirmation: (name: string) =>
    `Delete backup storage ${name}? It is removed once no instance, backup, restore or import uses it, and shows as Deleting until then. Its Secrets and the backup data in its bucket stay.`,
  cancel: 'Cancel',
};
