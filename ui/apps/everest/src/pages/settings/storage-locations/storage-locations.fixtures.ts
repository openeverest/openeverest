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

import {
  BackupStorageClass,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';

export const s3Class: BackupStorageClass = {
  metadata: { name: 's3' },
  spec: {
    displayName: 'S3-compatible object storage',
    description: 'Amazon S3 and S3-compatible object stores',
    authModes: [
      {
        name: 'AccessKey',
        credentials: {
          required: ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'],
        },
      },
      { name: 'WorkloadIdentity' },
    ],
    uiSchema: {
      location: {
        components: {
          bucket: {
            uiType: 'text',
            path: 'location.bucket',
            fieldParams: { label: 'Bucket', placeholder: 'my-backups' },
            validation: { required: true },
          },
          prefix: {
            uiType: 'text',
            path: 'location.prefix',
            fieldParams: {
              label: 'Prefix',
              helperText: 'Optional path within the bucket',
            },
          },
        },
        componentsOrder: ['bucket', 'prefix'],
      },
      storage: {
        components: {
          endpointURL: {
            uiType: 'text',
            path: 'parameters.endpointURL',
            fieldParams: { label: 'Endpoint URL' },
            validation: { required: true },
          },
          forcePathStyle: {
            uiType: 'toggle',
            path: 'parameters.forcePathStyle',
            fieldParams: { label: 'Force path-style URLs' },
          },
          implementation: {
            uiType: 'select',
            path: 'parameters.implementation',
            fieldParams: {
              label: 'Implementation',
              options: [
                { label: 'Amazon S3', value: 'aws' },
                { label: 'MinIO', value: 'minio' },
              ],
            },
          },
          region: {
            uiType: 'text',
            path: 'parameters.region',
            fieldParams: { label: 'Region' },
            validation: { required: true },
          },
        },
        componentsOrder: [
          'region',
          'endpointURL',
          'implementation',
          'forcePathStyle',
        ],
      },
      credentials: {
        AccessKey: {
          components: {
            AWS_ACCESS_KEY_ID: {
              uiType: 'text',
              path: 'AWS_ACCESS_KEY_ID',
              fieldParams: { label: 'Access key ID' },
              validation: { required: true },
            },
            AWS_SECRET_ACCESS_KEY: {
              uiType: 'text',
              path: 'AWS_SECRET_ACCESS_KEY',
              fieldParams: { label: 'Secret access key', type: 'password' },
              validation: { required: true },
            },
          },
          componentsOrder: ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'],
        },
      },
    } as unknown as BackupStorageClass['spec']['uiSchema'],
  },
};

export const s3Storage = (
  spec: Partial<BackupStorageCRD['spec']> = {},
  status?: BackupStorageCRD['status']
): BackupStorageCRD => ({
  metadata: { name: 's3-prod', namespace: 'team-a' },
  spec: {
    backupStorageClassRef: { name: 's3' },
    location: { bucket: 'my-backups', prefix: 'team-a/prod' },
    parameters: {
      region: 'us-east-1',
      endpointURL: 'https://s3.us-east-1.amazonaws.com',
      sessionToken: 'kept',
    } as unknown as BackupStorageCRD['spec']['parameters'],
    authentication: {
      mode: 'AccessKey',
      credentialsSecretRef: { name: 'backup-storage-s3-prod-credentials' },
    },
    ...spec,
  },
  ...(status && { status }),
});
