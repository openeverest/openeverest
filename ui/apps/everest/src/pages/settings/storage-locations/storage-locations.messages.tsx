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
    enterCredentials: 'Enter credentials',
    savedCredentials: 'Use saved credentials',
    currentCredentials: 'Current credentials',
    otherSavedCredentials: 'Use other saved credentials',
    credentialsSelect: 'Saved credentials',
    noCredentialsForm:
      'This authentication mode can only use saved credentials.',
    configured: 'Configured',
    replace: 'Replace',
    keepCurrentValue: 'Keep current value',
    optionalKeepsValue: 'Optional · leave blank to keep the current value',
    remove: 'Remove',
    removedOnSave: 'Removed when you save',
    undo: 'Undo',
    replaceAppliesEverywhere:
      'A replaced value applies to everything that uses these credentials.',
    credentialsUnreadable: (name: string) =>
      `Credentials ${name} can't be read: they are missing, or were saved outside OpenEverest. Use other saved credentials to change them.`,
    credentialsManagedOutside:
      'These credentials are managed outside OpenEverest; change them at their source.',
    credentialsNotUpdatable: (name: string) =>
      `You don't have permission to change credentials ${name}. Ask an administrator, or use other saved credentials.`,
    saveCredentialsAs: 'Save credentials as',
    saveCredentialsAsHelper: (defaultName: string) =>
      `Defaults to ${defaultName}.`,
    none: 'None',
    advanced: 'Advanced',
    tls: 'TLS certificate verification',
    tlsSystem: 'System trust store',
    tlsCustomCA: 'Trusted certificates',
    tlsSkipVerify: 'Skip certificate verification',
    tlsSkipVerifyWarning:
      "The server's certificate won't be verified, leaving connections to the storage open to man-in-the-middle attacks.",
    caPem: 'Paste certificates',
    caSaved: 'Use saved certificates',
    caExternal: 'Certificates from another tool, such as cert-manager',
    caBundle: 'Certificates (PEM)',
    caReplacesOwn:
      'Replaces the certificates saved for this storage, and for everything else that uses them.',
    caSelect: 'Saved certificates',
    caExternalName: 'Secret name',
    caKey: 'Key in the Secret',
    invalidCaBundle: 'Enter one or more PEM-encoded certificates',
    required: 'Required',
    secretConflict: (kind: BackupStorageSecretKind, secretName: string) =>
      kind === BackupStorageSecretKind.CA
        ? `Saved certificates named "${secretName}" already exist in this namespace.`
        : `Saved credentials named "${secretName}" already exist in this namespace.`,
    secretTaken: (kind: BackupStorageSecretKind, secretName: string) =>
      kind === BackupStorageSecretKind.CA
        ? `"${secretName}" already exists in this namespace and doesn't hold saved certificates. Use saved certificates instead.`
        : `"${secretName}" already exists in this namespace and doesn't hold credentials for this storage type and authentication mode. Use saved credentials, or save the new ones under another name in Advanced.`,
    useSecret: 'Use them',
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
