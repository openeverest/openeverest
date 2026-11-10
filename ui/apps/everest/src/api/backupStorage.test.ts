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

const SECRETS_URL = 'clusters/main/namespaces/team-a/secrets';
const STORAGES_URL = 'clusters/main/namespaces/team-a/backup-storages';
const MERGE_PATCH = {
  headers: { 'Content-Type': 'application/merge-patch+json' },
};
const DRY_RUN_MERGE_PATCH = { ...MERGE_PATCH, params: { dryRun: 'All' } };

const formValues = (
  overrides: Partial<BackupStorageFormValues> = {}
): BackupStorageFormValues => ({
  name: 's3-prod',
  namespace: 'team-a',
  backupStorageClass: 's3',
  spec: {
    location: { objectStore: { bucket: 'my-backups', prefix: 'team-a/prod' } },
    parameters: { region: 'us-east-1', forcePathStyle: false },
  },
  authMode: 'AccessKey',
  credentialsSource: CredentialsSource.New,
  credentialsSecretName: '',
  credentialsSaveAs: '',
  credentials: { AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'key' },
  replacedCredentialKeys: [],
  removedCredentialKeys: [],
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
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.get).mockImplementation((url) =>
    Promise.resolve({
      data: {
        metadata: { name: url.split('/').pop(), resourceVersion: '41' },
      },
    })
  );
  vi.mocked(api.post).mockImplementation((_url, body) =>
    Promise.resolve({ data: body })
  );
  vi.mocked(api.patch).mockImplementation((_url, body) =>
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
          metadata: {
            name: 'backup-storage-s3-prod-credentials',
            labels: {
              'openeverest.io/backup-storage-class': 's3',
              'openeverest.io/definition': 'AccessKey',
            },
          },
          type: 'Opaque',
          stringData: { AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'key' },
        },
        expect.anything(),
      ],
      [
        SECRETS_URL,
        {
          metadata: {
            name: 'backup-storage-s3-prod-ca',
            labels: { 'openeverest.io/definition': 'ca-bundle' },
          },
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
            location: {
              objectStore: { bucket: 'my-backups', prefix: 'team-a/prod' },
            },
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

  it('saves the credentials under the name entered in Advanced', async () => {
    await createBackupStorageFn(
      'main',
      formValues({ credentialsSaveAs: 'team-a-s3' })
    );

    expect(vi.mocked(api.post).mock.calls[0][1]).toMatchObject({
      metadata: { name: 'team-a-s3' },
    });
    expect(vi.mocked(api.post).mock.calls[1][1]).toMatchObject({
      spec: {
        authentication: { credentialsSecretRef: { name: 'team-a-s3' } },
      },
    });
  });

  it('references a saved CA bundle under ca.crt', async () => {
    await createBackupStorageFn(
      'main',
      formValues({
        tlsMode: TlsMode.CustomCA,
        caSource: CaSource.Existing,
        caSecretName: 'corp-ca',
        caSecretKey: 'bundle.pem',
      })
    );

    expect(api.post).toHaveBeenCalledTimes(2);
    expect(vi.mocked(api.post).mock.calls[1][1]).toMatchObject({
      spec: {
        tls: { ca: { secretKeyRef: { name: 'corp-ca', key: 'ca.crt' } } },
      },
    });
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
      location: { objectStore: { bucket: 'my-backups' } },
      parameters: { region: 'eu-west-1', implementation: 'minio' },
      authentication: {
        mode: 'AccessKey',
        credentialsSecretRef: { name: 'backup-storage-s3-prod-credentials' },
      },
      tls: { insecureSkipVerify: true },
    },
  } as unknown as BackupStorageCRD;

  const current = (
    overrides: Partial<BackupStorageFormValues> = {}
  ): BackupStorageFormValues =>
    formValues({
      credentialsSource: CredentialsSource.Current,
      credentialsSecretName: 'backup-storage-s3-prod-credentials',
      credentials: {},
      ...overrides,
    });

  const CREDENTIALS_URL = `${SECRETS_URL}/backup-storage-s3-prod-credentials`;
  const STORAGE_URL = `${STORAGES_URL}/s3-prod`;

  it('replaces the touched keys in place after a dry run', async () => {
    await editBackupStorageFn(
      'main',
      current({
        credentials: { AWS_SECRET_ACCESS_KEY: 'rotated' },
        removedCredentialKeys: ['AWS_SESSION_TOKEN'],
      }),
      previous
    );

    const patch = {
      metadata: { resourceVersion: '41' },
      stringData: { AWS_SECRET_ACCESS_KEY: 'rotated' },
      data: { AWS_SESSION_TOKEN: null },
    };
    expect(vi.mocked(api.get).mock.calls).toEqual([
      [CREDENTIALS_URL, { disableNotifications: false }],
    ]);
    expect(vi.mocked(api.patch).mock.calls).toEqual([
      [CREDENTIALS_URL, patch, DRY_RUN_MERGE_PATCH],
      [
        STORAGE_URL,
        expect.objectContaining({
          spec: expect.objectContaining({
            authentication: {
              mode: 'AccessKey',
              credentialsSecretRef: {
                name: 'backup-storage-s3-prod-credentials',
              },
            },
          }),
        }),
        MERGE_PATCH,
      ],
      [CREDENTIALS_URL, patch, MERGE_PATCH],
    ]);
    expect(api.post).not.toHaveBeenCalled();
  });

  it('leaves the credentials alone when no key is touched', async () => {
    await editBackupStorageFn('main', current(), previous);

    expect(api.get).not.toHaveBeenCalled();
    expect(api.post).not.toHaveBeenCalled();
    expect(vi.mocked(api.patch).mock.calls.map(([url]) => url)).toEqual([
      STORAGE_URL,
    ]);
  });

  it('writes nothing when the dry run is refused', async () => {
    vi.mocked(api.patch).mockRejectedValueOnce(new Error('forbidden'));

    await expect(
      editBackupStorageFn(
        'main',
        current({
          credentials: { AWS_SECRET_ACCESS_KEY: 'rotated' },
          tlsMode: TlsMode.CustomCA,
          caBundle: 'PEM',
        }),
        previous
      )
    ).rejects.toThrow('forbidden');

    expect(api.patch).toHaveBeenCalledTimes(1);
    expect(api.post).not.toHaveBeenCalled();
  });

  it('keeps the Secrets as they were when the storage is rejected', async () => {
    vi.mocked(api.patch).mockImplementation((url, body) =>
      url === STORAGE_URL
        ? Promise.reject(new Error('invalid'))
        : Promise.resolve({ data: body })
    );

    await expect(
      editBackupStorageFn(
        'main',
        current({
          credentials: { AWS_SECRET_ACCESS_KEY: 'rotated' },
          tlsMode: TlsMode.CustomCA,
          caBundle: 'PEM',
        }),
        previous
      )
    ).rejects.toThrow('invalid');

    expect(
      vi.mocked(api.patch).mock.calls.map(([url, , config]) => [url, config])
    ).toEqual([
      [CREDENTIALS_URL, DRY_RUN_MERGE_PATCH],
      [STORAGE_URL, MERGE_PATCH],
    ]);
    expect(vi.mocked(api.delete).mock.calls).toEqual([
      [`${SECRETS_URL}/backup-storage-s3-prod-ca`],
    ]);
  });

  it("replaces the storage's own pasted CA in place", async () => {
    const withOwnCa = {
      ...previous,
      spec: {
        ...previous.spec,
        tls: {
          ca: {
            secretKeyRef: { name: 'backup-storage-s3-prod-ca', key: 'ca.crt' },
          },
        },
      },
    };

    await editBackupStorageFn(
      'main',
      current({ tlsMode: TlsMode.CustomCA, caBundle: 'NEW PEM' }),
      withOwnCa
    );

    const patch = {
      metadata: { resourceVersion: '41' },
      stringData: { 'ca.crt': 'NEW PEM' },
    };
    const CA_URL = `${SECRETS_URL}/backup-storage-s3-prod-ca`;
    expect(api.post).not.toHaveBeenCalled();
    expect(vi.mocked(api.patch).mock.calls).toEqual([
      [CA_URL, patch, DRY_RUN_MERGE_PATCH],
      [
        STORAGE_URL,
        expect.objectContaining({
          spec: expect.objectContaining({
            tls: {
              ca: {
                secretKeyRef: {
                  name: 'backup-storage-s3-prod-ca',
                  key: 'ca.crt',
                },
              },
              insecureSkipVerify: null,
            },
          }),
        }),
        MERGE_PATCH,
      ],
      [CA_URL, patch, MERGE_PATCH],
    ]);
  });

  it('saves new credentials under the derived name on a mode change', async () => {
    await editBackupStorageFn(
      'main',
      formValues({ authMode: 'SessionKey' }),
      previous
    );

    expect(api.post).toHaveBeenCalledWith(
      SECRETS_URL,
      expect.objectContaining({
        metadata: {
          name: 'backup-storage-s3-prod-credentials',
          labels: {
            'openeverest.io/backup-storage-class': 's3',
            'openeverest.io/definition': 'SessionKey',
          },
        },
      }),
      expect.anything()
    );
    expect(api.patch).toHaveBeenCalledWith(
      STORAGE_URL,
      {
        spec: {
          parameters: {
            region: 'us-east-1',
            implementation: null,
            forcePathStyle: false,
          },
          authentication: {
            mode: 'SessionKey',
            credentialsSecretRef: {
              name: 'backup-storage-s3-prod-credentials',
            },
          },
          tls: null,
        },
      },
      MERGE_PATCH
    );
  });

  it('switches to other saved credentials and a CA from another tool', async () => {
    await editBackupStorageFn(
      'main',
      formValues({
        credentialsSource: CredentialsSource.Existing,
        credentialsSecretName: 'shared-s3',
        tlsMode: TlsMode.CustomCA,
        caSource: CaSource.External,
        caSecretName: 'corp-ca',
        caSecretKey: 'bundle.pem',
      }),
      previous
    );

    expect(api.post).not.toHaveBeenCalled();
    expect(api.get).not.toHaveBeenCalled();
    expect(api.patch).toHaveBeenCalledTimes(1);
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
