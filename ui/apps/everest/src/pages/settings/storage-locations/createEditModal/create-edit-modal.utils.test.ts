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
  BackupStorageFormValues,
  CaSource,
  CredentialsSource,
  TlsMode,
} from 'shared-types/backupStorages.types';
import { s3Class, s3Storage } from '../storage-locations.fixtures';
import {
  getCredentialsFields,
  getStorageFields,
  prepareStorageSubmitValues,
  storageLocationDefaultValues,
  storageLocationEditValues,
  storageLocationSchema,
} from './create-edit-modal.utils';

const PEM = '-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n';

// The s3 class with an optional key its form leaves out.
const s3ClassWithOptionalKey = {
  ...s3Class,
  spec: {
    ...s3Class.spec,
    authModes: [
      {
        name: 'AccessKey',
        credentials: {
          required: ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'],
          optional: ['AWS_SESSION_TOKEN'],
        },
      },
    ],
  },
};

const createValues = (
  overrides: Partial<BackupStorageFormValues> = {}
): BackupStorageFormValues => ({
  ...storageLocationDefaultValues('team-a'),
  name: 's3-prod',
  backupStorageClass: 's3',
  spec: {
    location: { objectStore: { bucket: ' my-backups ', prefix: '' } },
    parameters: {
      region: 'us-east-1',
      endpointURL: 'https://minio.local',
      implementation: '',
      forcePathStyle: true,
    },
  },
  authMode: 'AccessKey',
  credentials: { AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'key' },
  ...overrides,
});

const issuePaths = (
  values: BackupStorageFormValues,
  previous?: ReturnType<typeof s3Storage>
) => {
  const result = storageLocationSchema([s3Class], previous).safeParse(values);
  return result.success
    ? []
    : result.error.issues.map((issue) => issue.path.join('.'));
};

describe('class fields', () => {
  it('reads the location, storage and credentials sections in order', () => {
    const { location, storage } = getStorageFields(s3Class);

    expect(location.map(({ path, required }) => [path, required])).toEqual([
      ['location.objectStore.bucket', true],
      ['location.objectStore.prefix', false],
    ]);
    expect(storage.map(({ path, uiType }) => [path, uiType])).toEqual([
      ['parameters.region', 'text'],
      ['parameters.endpointURL', 'text'],
      ['parameters.implementation', 'select'],
      ['parameters.forcePathStyle', 'toggle'],
    ]);
    expect(storage[2].options).toHaveLength(2);
    expect(
      getCredentialsFields(s3Class, 'AccessKey').map(({ path, password }) => [
        path,
        password,
      ])
    ).toEqual([
      ['AWS_ACCESS_KEY_ID', false],
      ['AWS_SECRET_ACCESS_KEY', true],
    ]);
    expect(getCredentialsFields(s3Class, 'WorkloadIdentity')).toEqual([]);
  });

  it('takes the keys and which are required from the auth mode', () => {
    expect(
      getCredentialsFields(s3ClassWithOptionalKey, 'AccessKey').map(
        ({ path, label, password, required, requiredByMode }) => ({
          path,
          label,
          password,
          required,
          requiredByMode,
        })
      )
    ).toEqual([
      {
        path: 'AWS_ACCESS_KEY_ID',
        label: 'Access key ID',
        password: false,
        required: true,
        requiredByMode: true,
      },
      {
        path: 'AWS_SECRET_ACCESS_KEY',
        label: 'Secret access key',
        password: true,
        required: true,
        requiredByMode: true,
      },
      {
        path: 'AWS_SESSION_TOKEN',
        label: 'AWS_SESSION_TOKEN',
        password: true,
        required: false,
        requiredByMode: false,
      },
    ]);
  });
});

describe('storageLocationEditValues', () => {
  it('maps a storage with an unlabelled CA Secret onto the form', () => {
    const storage = s3Storage({
      tls: { ca: { secretKeyRef: { name: 'corp-ca', key: 'bundle.pem' } } },
    });

    expect(storageLocationEditValues(storage)).toEqual({
      name: 's3-prod',
      namespace: 'team-a',
      backupStorageClass: 's3',
      spec: {
        location: {
          objectStore: { bucket: 'my-backups', prefix: 'team-a/prod' },
        },
        parameters: storage.spec.parameters,
      },
      authMode: 'AccessKey',
      credentialsSource: CredentialsSource.Current,
      credentialsSecretName: 'backup-storage-s3-prod-credentials',
      credentialsSaveAs: '',
      credentials: {},
      replacedCredentialKeys: [],
      removedCredentialKeys: [],
      tlsMode: TlsMode.CustomCA,
      caSource: CaSource.External,
      caBundle: '',
      caSecretName: 'corp-ca',
      caSecretKey: 'bundle.pem',
    });
  });

  it('maps a CA bundle under ca.crt to saved certificates', () => {
    expect(
      storageLocationEditValues(
        s3Storage({
          tls: { ca: { secretKeyRef: { name: 'corp-ca', key: 'ca.crt' } } },
        })
      )
    ).toMatchObject({ caSource: CaSource.Existing, caSecretName: 'corp-ca' });
  });

  it('enters credentials for a storage that has none', () => {
    expect(
      storageLocationEditValues(
        s3Storage({ authentication: { mode: 'WorkloadIdentity' } })
      ).credentialsSource
    ).toBe(CredentialsSource.New);
  });

  it('maps skipped verification and the system trust store', () => {
    expect(
      storageLocationEditValues(
        s3Storage({ tls: { insecureSkipVerify: true } })
      ).tlsMode
    ).toBe(TlsMode.SkipVerify);
    expect(storageLocationEditValues(s3Storage()).tlsMode).toBe(TlsMode.System);
  });
});

describe('prepareStorageSubmitValues', () => {
  it('keeps only what the class renders and drops empty values', () => {
    const values = createValues({
      spec: {
        location: { objectStore: { bucket: ' my-backups ', prefix: '' } },
        parameters: {
          region: 'us-east-1',
          endpointURL: 'https://minio.local',
          implementation: '',
          forcePathStyle: true,
          fromAnotherClass: 'x',
        },
      },
      credentials: {
        AWS_ACCESS_KEY_ID: 'id',
        AWS_SECRET_ACCESS_KEY: 'key',
        STALE: 'y',
      },
    });

    expect(prepareStorageSubmitValues(values, s3Class)).toEqual({
      ...values,
      spec: {
        location: { objectStore: { bucket: 'my-backups' } },
        parameters: {
          region: 'us-east-1',
          endpointURL: 'https://minio.local',
          forcePathStyle: true,
        },
      },
      credentialsSource: CredentialsSource.New,
      credentialsSecretName: '',
      credentialsSaveAs: '',
      credentials: { AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'key' },
      replacedCredentialKeys: [],
      removedCredentialKeys: [],
    });
  });

  it('keeps the name new credentials are saved as', () => {
    expect(
      prepareStorageSubmitValues(
        createValues({ credentialsSaveAs: ' team-a-s3 ' }),
        s3Class
      ).credentialsSaveAs
    ).toBe('team-a-s3');
  });

  it('keeps the current Secret and unrendered parameters on edit', () => {
    const previous = s3Storage();
    const values = storageLocationEditValues(previous);
    values.spec.parameters = {
      ...values.spec.parameters,
      region: 'eu-west-1',
      implementation: '',
    };

    const prepared = prepareStorageSubmitValues(values, s3Class, previous);

    expect(prepared.spec.parameters).toEqual({
      region: 'eu-west-1',
      endpointURL: 'https://s3.us-east-1.amazonaws.com',
      sessionToken: 'kept',
    });
    expect(prepared).toMatchObject({
      credentialsSource: CredentialsSource.Current,
      credentialsSecretName: 'backup-storage-s3-prod-credentials',
      credentials: {},
      removedCredentialKeys: [],
    });
  });

  it('sends only the keys replaced or removed on edit', () => {
    const previous = s3Storage();
    const values = {
      ...storageLocationEditValues(previous),
      credentials: {
        AWS_ACCESS_KEY_ID: 'typed, then kept',
        AWS_SECRET_ACCESS_KEY: 'rotated',
        AWS_SESSION_TOKEN: 'typed, then removed',
      },
      replacedCredentialKeys: ['AWS_SECRET_ACCESS_KEY'],
      removedCredentialKeys: ['AWS_SESSION_TOKEN'],
    };

    expect(
      prepareStorageSubmitValues(values, s3ClassWithOptionalKey, previous)
    ).toMatchObject({
      credentialsSource: CredentialsSource.Current,
      credentials: { AWS_SECRET_ACCESS_KEY: 'rotated' },
      removedCredentialKeys: ['AWS_SESSION_TOKEN'],
    });
    expect(
      prepareStorageSubmitValues(
        {
          ...values,
          removedCredentialKeys: [],
          credentials: { AWS_SESSION_TOKEN: 'token' },
        },
        s3ClassWithOptionalKey,
        previous
      )
    ).toMatchObject({
      credentials: { AWS_SESSION_TOKEN: 'token' },
      removedCredentialKeys: [],
    });
  });

  it('enters new credentials on edit once the auth mode changes', () => {
    const previous = s3Storage({
      authentication: { mode: 'WorkloadIdentity' },
    });
    const values = {
      ...storageLocationEditValues(previous),
      authMode: 'AccessKey',
      credentials: { AWS_ACCESS_KEY_ID: 'id2', AWS_SECRET_ACCESS_KEY: 'key2' },
    };

    const prepared = prepareStorageSubmitValues(values, s3Class, previous);

    expect(prepared.credentialsSource).toBe(CredentialsSource.New);
    expect(prepared.credentials).toEqual(values.credentials);
  });

  it('uses the chosen Secret, or none for a mode without credentials', () => {
    expect(
      prepareStorageSubmitValues(
        createValues({
          credentialsSource: CredentialsSource.Existing,
          credentialsSecretName: 'team-creds',
        }),
        s3Class
      )
    ).toMatchObject({
      credentialsSource: CredentialsSource.Existing,
      credentialsSecretName: 'team-creds',
      credentials: {},
    });
    expect(
      prepareStorageSubmitValues(
        createValues({ authMode: 'WorkloadIdentity' }),
        s3Class
      )
    ).toMatchObject({
      credentialsSource: CredentialsSource.Existing,
      credentialsSecretName: '',
      credentials: {},
    });
  });
});

describe('storageLocationSchema', () => {
  it('accepts a complete form', () => {
    expect(issuePaths(createValues())).toEqual([]);
  });

  it("requires the class's required fields and the credentials", () => {
    expect(
      issuePaths(
        createValues({
          spec: {
            location: { objectStore: { bucket: '' } },
            parameters: { region: '' },
          },
          credentials: { AWS_ACCESS_KEY_ID: 'id' },
        })
      )
    ).toEqual([
      'spec.location.objectStore.bucket',
      'spec.parameters.region',
      'spec.parameters.endpointURL',
      'credentials.AWS_SECRET_ACCESS_KEY',
    ]);
  });

  it('requires a Secret when not entering credentials', () => {
    expect(
      issuePaths(
        createValues({ credentialsSource: CredentialsSource.Existing })
      )
    ).toEqual(['credentialsSecretName']);
    expect(
      issuePaths(
        createValues({ authMode: 'WorkloadIdentity', credentials: {} })
      )
    ).toEqual([]);
  });

  it('requires a value only for the keys being replaced on edit', () => {
    const previous = s3Storage();

    expect(issuePaths(storageLocationEditValues(previous), previous)).toEqual(
      []
    );
    expect(
      issuePaths(
        {
          ...storageLocationEditValues(previous),
          replacedCredentialKeys: ['AWS_SECRET_ACCESS_KEY'],
        },
        previous
      )
    ).toEqual(['credentials.AWS_SECRET_ACCESS_KEY']);
  });

  it('checks the name new credentials are saved as', () => {
    expect(
      issuePaths(createValues({ credentialsSaveAs: 'team-a-s3' }))
    ).toEqual([]);
    expect(
      issuePaths(createValues({ credentialsSaveAs: 'Not A Name' }))
    ).toContain('credentialsSaveAs');
  });

  it('validates the custom CA', () => {
    const customCA = { tlsMode: TlsMode.CustomCA };

    expect(issuePaths(createValues({ ...customCA, caBundle: 'nope' }))).toEqual(
      ['caBundle']
    );
    expect(issuePaths(createValues({ ...customCA, caBundle: PEM }))).toEqual(
      []
    );
    expect(
      issuePaths(createValues({ ...customCA, caBundle: `\n${PEM}\n${PEM}` }))
    ).toEqual([]);
    expect(
      issuePaths(
        createValues({ ...customCA, caBundle: PEM.replace('MIIB', 'MI!B') })
      )
    ).toEqual(['caBundle']);
    // Backtracks exponentially with a body that may span END/BEGIN lines.
    const pathological = `-----BEGIN CERTIFICATE-----${'a-----END CERTIFICATE----------BEGIN CERTIFICATE-----'.repeat(30)}!`;
    expect(
      issuePaths(createValues({ ...customCA, caBundle: pathological }))
    ).toEqual(['caBundle']);
    expect(
      issuePaths(
        createValues({
          ...customCA,
          caSource: CaSource.Existing,
          caSecretKey: '',
        })
      )
    ).toEqual(['caSecretName']);
    expect(
      issuePaths(
        createValues({
          ...customCA,
          caSource: CaSource.External,
          caSecretKey: '',
        })
      )
    ).toEqual(['caSecretName', 'caSecretKey']);
  });
});
