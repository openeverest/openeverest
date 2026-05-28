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
  createEditModal: {
    addEditModal: (isEditMode: boolean) =>
      `${isEditMode ? 'Edit' : 'Add'} Backup storage`,
    addEditButton: (isEditMode: boolean) => (isEditMode ? 'Edit' : 'Add'),
    cancel: 'Cancel',
    placeholders: {
      name: 'Enter storage display name',
      namespace: 'Select namespace',
      type: 'Enter bucket name',
      region: 'Enter region',
      url: 'Enter URL',
      accessKey: 'Enter Access key',
      secretKey: 'Enter Secret key',
    },
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
  s3: 'S3 Compatible',
  gcs: 'Google Cloud Storage',
  azure: 'Azure Cloud Storage',
  name: 'Name',
  namespace: 'Namespace',
  type: 'Type',
  bucketName: 'Bucket Name',
  region: 'Region',
  url: 'Endpoint',
  accessKey: 'Access Key',
  secretKey: 'Secret Key',
  verifyTLS: 'Verify TLS certificate',
  forcePathStyle: 'Force path-style URL access',
  edit: 'Edit',
  delete: 'Delete',
  addStorageLocationButton: 'Add backup storage',
  noData: "No backup storage set up yet. Let's get started!",
};
