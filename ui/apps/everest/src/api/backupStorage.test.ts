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

import { api } from 'api/api';
import {
  BackupStorageCRD,
  BackupStorageFormValues,
  BackupStorageSecretKind,
  CaSource,
  CredentialsSource,
  TlsMode,
} from 'shared-types/backupStorages.types';
import {
  BackupStorageSecretConflictError,
  createBackupStorageFn,
  editBackupStorageFn,
} from './backupStorage';

vi.mock('api/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));
vi.mock('utils/generateShortUID', () => ({ generateShortUID: () => 'x1y' }));

const SECRETS_URL = 'clusters/main/namespaces/team-a/secrets';
const STORAGES_URL = 'clusters/main/namespaces/team-a/backup-storages';

const formValues = (
  overrides: Partial<BackupStorageFormValues> = {}
): BackupStorageFormValues => ({
  name: 's3-prod',
  namespace: 'team-a',
  backupStorageClass: 's3',
  spec: {
    location: { bucket: 'my-backups', prefix: 'team-a/prod' },
    parameters: { region: 'us-east-1', forcePathStyle: false },
  },
  authMode: 'AccessKey',
  credentialsSource: CredentialsSource.New,
  credentialsSecretName: '',
  credentials: { AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'key' },
  tlsMode: TlsMode.System,
  caSource: CaSource.New,
  caBundle: '',
  caSecretName: '',
  caSecretKey: '',
  ...overrides,
});

const conflict = () =>
  Promise.reject(Object.assign(new Error('conflict'), { status: 409 }));

beforeEach(() => {
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.post).mockImplementation((_url, body) =>
    Promise.resolve({ data: body })
  );
  vi.mocked(api.delete).mockResolvedValue({ data: {} });
});

describe('createBackupStorageFn', () => {
  it('creates the credentials and CA Secrets, then the storage', async () => {
    await createBackupStorageFn(
      'main',
      formValues({ tlsMode: TlsMode.CustomCA, caBundle: 'PEM' })
    );

    expect(vi.mocked(api.post).mock.calls).toEqual([
      [
        SECRETS_URL,
        {
          metadata: { name: 'backup-storage-s3-prod-credentials' },
          type: 'Opaque',
          stringData: { AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'key' },
        },
        expect.anything(),
      ],
      [
        SECRETS_URL,
        {
          metadata: { name: 'backup-storage-s3-prod-ca' },
          type: 'Opaque',
          stringData: { 'ca.crt': 'PEM' },
        },
        expect.anything(),
      ],
      [
        STORAGES_URL,
        {
          metadata: { name: 's3-prod', namespace: 'team-a' },
          spec: {
            backupStorageClassRef: { name: 's3' },
            location: { bucket: 'my-backups', prefix: 'team-a/prod' },
            parameters: { region: 'us-east-1', forcePathStyle: false },
            authentication: {
              mode: 'AccessKey',
              credentialsSecretRef: {
                name: 'backup-storage-s3-prod-credentials',
              },
            },
            tls: {
              ca: {
                secretKeyRef: {
                  name: 'backup-storage-s3-prod-ca',
                  key: 'ca.crt',
                },
              },
            },
          },
        },
      ],
    ]);
  });

  it('references an existing Secret without creating one', async () => {
    await createBackupStorageFn(
      'main',
      formValues({
        credentialsSource: CredentialsSource.Existing,
        credentialsSecretName: 'shared-s3',
        tlsMode: TlsMode.SkipVerify,
      })
    );

    expect(api.post).toHaveBeenCalledTimes(1);
    expect(api.post).toHaveBeenCalledWith(
      STORAGES_URL,
      expect.objectContaining({
        spec: expect.objectContaining({
          authentication: {
            mode: 'AccessKey',
            credentialsSecretRef: { name: 'shared-s3' },
          },
          tls: { insecureSkipVerify: true },
        }),
      })
    );
  });

  it('reports a taken credentials Secret name', async () => {
    vi.mocked(api.post).mockImplementationOnce(conflict);

    const error = await createBackupStorageFn('main', formValues()).catch(
      (e) => e
    );

    expect(error).toBeInstanceOf(BackupStorageSecretConflictError);
    expect(error).toMatchObject({
      kind: BackupStorageSecretKind.Credentials,
      secretName: 'backup-storage-s3-prod-credentials',
    });
    expect(api.post).toHaveBeenCalledTimes(1);
    expect(api.delete).not.toHaveBeenCalled();
  });

  it('deletes the Secrets it created when the storage is rejected', async () => {
    vi.mocked(api.post)
      .mockResolvedValueOnce({ data: {} })
      .mockResolvedValueOnce({ data: {} })
      .mockRejectedValueOnce(new Error('invalid'));

    await expect(
      createBackupStorageFn(
        'main',
        formValues({ tlsMode: TlsMode.CustomCA, caBundle: 'PEM' })
      )
    ).rejects.toThrow('invalid');

    expect(vi.mocked(api.delete).mock.calls).toEqual([
      [`${SECRETS_URL}/backup-storage-s3-prod-credentials`],
      [`${SECRETS_URL}/backup-storage-s3-prod-ca`],
    ]);
  });

  it('deletes the credentials Secret when the CA Secret fails', async () => {
    vi.mocked(api.post)
      .mockResolvedValueOnce({ data: {} })
      .mockImplementationOnce(conflict);

    await expect(
      createBackupStorageFn(
        'main',
        formValues({ tlsMode: TlsMode.CustomCA, caBundle: 'PEM' })
      )
    ).rejects.toMatchObject({ kind: BackupStorageSecretKind.CA });

    expect(vi.mocked(api.delete).mock.calls).toEqual([
      [`${SECRETS_URL}/backup-storage-s3-prod-credentials`],
    ]);
  });
});

describe('editBackupStorageFn', () => {
  const previous = {
    metadata: { name: 's3-prod', namespace: 'team-a' },
    spec: {
      backupStorageClassRef: { name: 's3' },
      location: { bucket: 'my-backups' },
      parameters: { region: 'eu-west-1', implementation: 'minio' },
      authentication: {
        mode: 'AccessKey',
        credentialsSecretRef: { name: 'backup-storage-s3-prod-credentials' },
      },
      tls: { insecureSkipVerify: true },
    },
  } as unknown as BackupStorageCRD;

  it('writes new credentials under a unique name and patches the storage', async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: previous });

    await editBackupStorageFn('main', formValues(), previous);

    expect(api.post).toHaveBeenCalledWith(
      SECRETS_URL,
      expect.objectContaining({
        metadata: { name: 'backup-storage-s3-prod-credentials-x1y' },
      }),
      expect.anything()
    );
    expect(api.patch).toHaveBeenCalledWith(
      `${STORAGES_URL}/s3-prod`,
      {
        spec: {
          parameters: {
            region: 'us-east-1',
            implementation: null,
            forcePathStyle: false,
          },
          authentication: {
            mode: 'AccessKey',
            credentialsSecretRef: {
              name: 'backup-storage-s3-prod-credentials-x1y',
            },
          },
          tls: null,
        },
      },
      { headers: { 'Content-Type': 'application/merge-patch+json' } }
    );
  });

  it('switches to another existing Secret and a custom CA', async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: previous });

    await editBackupStorageFn(
      'main',
      formValues({
        credentialsSource: CredentialsSource.Existing,
        credentialsSecretName: 'shared-s3',
        tlsMode: TlsMode.CustomCA,
        caSource: CaSource.Existing,
        caSecretName: 'corp-ca',
        caSecretKey: 'bundle.pem',
      }),
      previous
    );

    expect(api.post).not.toHaveBeenCalled();
    expect(vi.mocked(api.patch).mock.calls[0][1]).toMatchObject({
      spec: {
        authentication: { credentialsSecretRef: { name: 'shared-s3' } },
        tls: {
          ca: { secretKeyRef: { name: 'corp-ca', key: 'bundle.pem' } },
          insecureSkipVerify: null,
        },
      },
    });
  });
});
