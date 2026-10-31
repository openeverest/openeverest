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

const createValues = (
  overrides: Partial<BackupStorageFormValues> = {}
): BackupStorageFormValues => ({
  ...storageLocationDefaultValues('team-a'),
  name: 's3-prod',
  backupStorageClass: 's3',
  spec: {
    location: { bucket: ' my-backups ', prefix: '' },
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
      ['location.bucket', true],
      ['location.prefix', false],
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
});

describe('storageLocationEditValues', () => {
  it('maps a storage with a CA Secret onto the form', () => {
    const storage = s3Storage({
      tls: { ca: { secretKeyRef: { name: 'corp-ca', key: 'bundle.pem' } } },
    });

    expect(storageLocationEditValues(storage)).toEqual({
      name: 's3-prod',
      namespace: 'team-a',
      backupStorageClass: 's3',
      spec: {
        location: { bucket: 'my-backups', prefix: 'team-a/prod' },
        parameters: storage.spec.parameters,
      },
      authMode: 'AccessKey',
      credentialsSource: CredentialsSource.New,
      credentialsSecretName: 'backup-storage-s3-prod-credentials',
      credentials: {},
      tlsMode: TlsMode.CustomCA,
      caSource: CaSource.Existing,
      caBundle: '',
      caSecretName: 'corp-ca',
      caSecretKey: 'bundle.pem',
    });
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
        location: { bucket: ' my-backups ', prefix: '' },
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
        location: { bucket: 'my-backups' },
        parameters: {
          region: 'us-east-1',
          endpointURL: 'https://minio.local',
          forcePathStyle: true,
        },
      },
      credentialsSource: CredentialsSource.New,
      credentialsSecretName: '',
      credentials: { AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'key' },
    });
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
    expect(prepared.credentialsSource).toBe(CredentialsSource.Existing);
    expect(prepared.credentialsSecretName).toBe(
      'backup-storage-s3-prod-credentials'
    );
    expect(prepared.credentials).toEqual({});
  });

  it('writes new credentials on edit once they are entered', () => {
    const previous = s3Storage();
    const values = {
      ...storageLocationEditValues(previous),
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
          spec: { location: { bucket: '' }, parameters: { region: '' } },
          credentials: { AWS_ACCESS_KEY_ID: 'id' },
        })
      )
    ).toEqual([
      'spec.location.bucket',
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

  it('lets an edit leave the credentials blank', () => {
    const previous = s3Storage();

    expect(issuePaths(storageLocationEditValues(previous), previous)).toEqual(
      []
    );
    expect(
      issuePaths(
        {
          ...storageLocationEditValues(previous),
          credentials: { AWS_ACCESS_KEY_ID: 'id2' },
        },
        previous
      )
    ).toEqual(['credentials.AWS_SECRET_ACCESS_KEY']);
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
      issuePaths(
        createValues({
          ...customCA,
          caSource: CaSource.Existing,
          caSecretKey: '',
        })
      )
    ).toEqual(['caSecretName', 'caSecretKey']);
  });
});
