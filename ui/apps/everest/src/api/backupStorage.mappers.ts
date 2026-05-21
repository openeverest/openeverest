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
  BackupStorage,
  BackupStorageCRD,
  StorageType,
} from 'shared-types/backupStorages.types';

type KubeMetadata = { name?: string; namespace?: string };

/**
 * Maps a BackupStorage CRD object (from the v2 API) to the flat UI model.
 * Credentials (accessKey/secretKey) are write-only on the CRD and will be empty
 * strings on read.
 */
export const crdToFlat = (crd: BackupStorageCRD): BackupStorage => {
  const meta = crd.metadata as unknown as KubeMetadata | undefined;
  return {
    name: meta?.name ?? '',
    namespace: meta?.namespace ?? '',
    type: crd.spec.type as StorageType,
    bucketName: crd.spec.s3?.bucket ?? '',
    url: crd.spec.s3?.endpointURL ?? '',
    region: crd.spec.s3?.region ?? '',
    accessKey: crd.spec.s3?.accessKeyId ?? '',
    secretKey: crd.spec.s3?.secretAccessKey ?? '',
    verifyTLS: crd.spec.s3?.verifyTLS ?? true,
    forcePathStyle: crd.spec.s3?.forcePathStyle ?? false,
  };
};

/**
 * Maps a flat UI model to a CRD payload for creating a new BackupStorage.
 */
export const flatToCrdCreate = (flat: BackupStorage) => ({
  metadata: {
    name: flat.name,
    namespace: flat.namespace,
  },
  spec: {
    type: flat.type,
    s3: {
      bucket: flat.bucketName,
      endpointURL: flat.url,
      region: flat.region,
      credentialsSecretName: `backup-storage-${flat.name}-credentials`,
      accessKeyId: flat.accessKey,
      secretAccessKey: flat.secretKey,
      verifyTLS: flat.verifyTLS,
      forcePathStyle: flat.forcePathStyle,
    },
  },
});

/**
 * Maps editable flat fields to a partial CRD payload for patching.
 * Name, namespace, and type are immutable and excluded.
 */
export const flatToCrdEdit = (flat: BackupStorage) => ({
  spec: {
    type: flat.type,
    s3: {
      bucket: flat.bucketName,
      endpointURL: flat.url,
      region: flat.region,
      credentialsSecretName: `backup-storage-${flat.name}-credentials`,
      ...(flat.accessKey && { accessKeyId: flat.accessKey }),
      ...(flat.secretKey && { secretAccessKey: flat.secretKey }),
      verifyTLS: flat.verifyTLS,
      forcePathStyle: flat.forcePathStyle,
    },
  },
});
