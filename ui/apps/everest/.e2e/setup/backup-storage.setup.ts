// everest
// Copyright (C) 2023 Percona LLC
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

import { expect, test as setup } from '@playwright/test';
import 'dotenv/config';
import { getCITokenFromLocalStorage } from '../utils/localStorage';
import {
  EVEREST_CI_CLUSTER,
  EVEREST_CI_NAMESPACES,
  getBucketNamespacesMap,
} from '../constants';

const {
  EVEREST_LOCATION_ACCESS_KEY,
  EVEREST_LOCATION_SECRET_KEY,
  EVEREST_LOCATION_REGION,
  EVEREST_LOCATION_URL,
  EVEREST_S3_ACCESS_KEY,
  EVEREST_S3_SECRET_KEY,
  EVEREST_S3_REGION,
  EVEREST_S3_URL,
} = process.env;

const postBackupStorage = async (
  request: any,
  token: string,
  namespace: string,
  data: any
) =>
  request.post(
    `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/backup-storages/`,
    {
      data,
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );

setup.describe.serial('Backup Storage setup', () => {
  setup('Create Backup Storages', async ({ request }) => {
    const token = await getCITokenFromLocalStorage();
    const promises: Promise<unknown>[] = [];
    // This has a nested array structure, in the form of
    // [
    //   ['bucket1', 'namespace1'],
    //   ['bucket2', 'namespace2'],
    // ]
    const bucketNamespacesMap = getBucketNamespacesMap();

    bucketNamespacesMap.forEach(async ([bucket, namespace]) => {
      const isEverestTesting = bucket === 'everest-testing';
      const data = isEverestTesting
        ? {
            metadata: {
              name: bucket,
              namespace,
            },
            spec: {
              type: 's3',
              s3: {
                bucket,
                endpointURL: EVEREST_S3_URL,
                region: EVEREST_S3_REGION,
                credentialsSecretRef: {
                  name: `backup-storage-${bucket}-credentials`,
                },
                accessKeyId: EVEREST_S3_ACCESS_KEY,
                secretAccessKey: EVEREST_S3_SECRET_KEY,
                verifyTLS: true,
                forcePathStyle: false,
              },
            },
          }
        : {
            metadata: {
              name: bucket,
              namespace,
            },
            spec: {
              type: 's3',
              s3: {
                bucket,
                endpointURL: EVEREST_LOCATION_URL,
                region: EVEREST_LOCATION_REGION,
                credentialsSecretRef: {
                  name: `backup-storage-${bucket}-credentials`,
                },
                accessKeyId: EVEREST_LOCATION_ACCESS_KEY,
                secretAccessKey: EVEREST_LOCATION_SECRET_KEY,
                verifyTLS: false,
                forcePathStyle: true,
              },
            },
          };

      promises.push(
        (async () => {
          let response = await postBackupStorage(
            request,
            token,
            namespace,
            data
          );

          if (response.status() === 404) {
            const body = await response.json();
            const isMissingNamespace =
              body?.message && body.message.includes('not found');

            if (isMissingNamespace) {
              response = await postBackupStorage(
                request,
                token,
                EVEREST_CI_NAMESPACES.EVEREST_UI,
                {
                  ...data,
                  metadata: {
                    ...data.metadata,
                    namespace: EVEREST_CI_NAMESPACES.EVEREST_UI,
                  },
                }
              );
            }
          }

          expect([200, 201, 409]).toContain(response.status());
        })()
      );
    });
    await Promise.all(promises);
  });
});
