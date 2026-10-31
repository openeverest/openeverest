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

import { BackupStorageSecretKind } from 'shared-types/backupStorages.types';

export const Messages = {
  createEditModal: {
    addEditModal: (isEditMode: boolean) =>
      `${isEditMode ? 'Edit' : 'Add'} Backup storage`,
    addEditButton: (isEditMode: boolean) => (isEditMode ? 'Edit' : 'Add'),
    cancel: 'Cancel',
    placeholders: {
      name: 'Enter storage display name',
      namespace: 'Select namespace',
    },
    noClasses: 'No backup storage class is available in this cluster',
    authentication: 'Authentication',
    authMode: 'Authentication mode',
    credentials: 'Credentials',
    enterCredentials: 'Enter credentials',
    existingSecret: 'Use an existing Secret',
    secret: 'Secret',
    keepCredentialsOnEdit:
      'Leave the fields blank to keep the current Secret. New credentials are saved to a new Secret.',
    noCredentialsForm:
      'This authentication mode can only use an existing Secret.',
    none: 'None',
    advanced: 'Advanced',
    tls: 'TLS certificate verification',
    tlsSystem: 'System trust store',
    tlsCustomCA: 'Custom CA',
    tlsSkipVerify: 'Skip certificate verification',
    tlsSkipVerifyWarning:
      "The server's certificate won't be verified, leaving connections to the storage open to man-in-the-middle attacks.",
    caPem: 'Paste a PEM bundle',
    caBundle: 'CA certificates (PEM)',
    caKey: 'Key',
    invalidCaBundle: 'Enter one or more PEM-encoded certificates',
    required: 'Required',
    secretConflict: (kind: BackupStorageSecretKind, secretName: string) =>
      `A Secret named "${secretName}" already exists in this namespace, so the ${
        kind === BackupStorageSecretKind.CA ? 'CA bundle' : 'credentials'
      } can't be saved under that name.`,
    useSecret: 'Use that Secret',
  },
  deleteDialog: {
    header: 'Delete storage',
    content: (storageName: string) => (
      <>
        Are you sure you want to permanently delete this backup storage{' '}
        <b>{storageName}</b>? Any database clusters using this storage will no
        longer be able to execute their scheduled backup jobs.
      </>
    ),
  },
  status: {
    accepted: 'Ready',
    deleting: 'Deleting',
    pending: 'Pending',
  },
  name: 'Name',
  namespace: 'Namespace',
  backupStorageClass: 'Class',
  bucket: 'Bucket',
  authMode: 'Auth mode',
  statusHeader: 'Status',
  credentialsSecret: 'Credentials Secret',
  caSecret: 'CA Secret',
  insecureSkipVerify: 'TLS verification',
  skipped: 'Skipped',
  systemTrustStore: 'System trust store',
  edit: 'Edit',
  delete: 'Delete',
  addStorageLocationButton: 'Add backup storage',
  noData: "No backup storage set up yet. Let's get started!",
};
