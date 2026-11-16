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
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError, AxiosHeaders } from 'axios';
import { createSecretFn, getSecretFn, patchSecretFn } from 'api/secrets';
import {
  createBackupStorageFn,
  getBackupStorageFn,
  updateBackupStorageFn,
} from 'api/backupStorage';
import { getBackupStorageClassesFn } from 'api/backupStorageClasses';
import {
  BackupStorageClass,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';
import { TestWrapper } from 'utils/test';
import { BackupStorageModal } from './backup-storage-modal';
import { BackupStorageTarget } from './backup-storage-modal.types';

vi.mock('api/secrets', () => ({
  createSecretFn: vi.fn(),
  deleteSecretFn: vi.fn(),
  getSecretFn: vi.fn(),
  getSecretsFn: vi.fn(),
  patchSecretFn: vi.fn(),
}));
vi.mock('api/backupStorage', () => ({
  createBackupStorageFn: vi.fn(),
  updateBackupStorageFn: vi.fn(),
  getBackupStorageFn: vi.fn(),
  getBackupStoragesFn: vi.fn(),
  deleteBackupStorageFn: vi.fn(),
}));
vi.mock('api/backupStorageClasses', () => ({
  getBackupStorageClassesFn: vi.fn(),
}));
const rbacMocks = vi.hoisted(() => ({
  // Checks secrets:use on one Secret, as per-Secret grants allow it.
  check: vi.fn<
    (action: string, resource: string, object: string) => Promise<boolean>
  >(async () => true),
}));
vi.mock('hooks/rbac', async (importOriginal) => {
  const namespacePermissions = {
    canCreate: ['team-a'],
    canRead: ['team-a'],
    canUpdate: ['team-a'],
    canDelete: ['team-a'],
    isPermissionsLoading: false,
  };
  const permissions = {
    canRead: true,
    canCreate: true,
    canUpdate: true,
    canDelete: true,
    isPermissionsLoading: false,
  };
  return {
    ...(await importOriginal<typeof import('hooks/rbac')>()),
    useNamespacePermissionsForResource: () => namespacePermissions,
    useRBACPermissions: () => permissions,
    useRBACCheck: () => rbacMocks.check,
  };
});

// The reference S3 class, its storage form trimmed to region and endpoint
// so that forcePathStyle is a parameter the form doesn't show.
const s3: BackupStorageClass = {
  metadata: { name: 's3' },
  spec: {
    displayName: 'S3-compatible object storage',
    authModes: [
      // A storage can't use it, whatever the class lists.
      { name: 'Anonymous' },
      {
        name: 'AccessKey',
        credentials: {
          required: ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'],
        },
      },
    ],
    parametersSchema: {
      openAPIV3Schema: {
        type: 'object',
        required: ['endpointURL', 'region'],
        properties: {
          endpointURL: { type: 'string', pattern: '^https?://[^/?#@]+' },
          forcePathStyle: { type: 'boolean' },
          region: { type: 'string', minLength: 1, maxLength: 63 },
        },
      },
    },
  },
};
Reflect.set(s3.spec, 'uiSchema', {
  location: {
    components: {
      bucket: {
        path: 'location.objectStore.bucket',
        uiType: 'text',
        fieldParams: { label: 'Bucket' },
        validation: { required: true },
      },
      prefix: {
        path: 'location.objectStore.prefix',
        uiType: 'text',
        fieldParams: { label: 'Prefix' },
      },
    },
    componentsOrder: ['bucket', 'prefix'],
  },
  storage: {
    components: {
      region: {
        path: 'parameters.region',
        uiType: 'text',
        fieldParams: { label: 'Region' },
        validation: { required: true },
      },
      endpointURL: {
        path: 'parameters.endpointURL',
        uiType: 'text',
        fieldParams: { label: 'Endpoint URL' },
        validation: { required: true },
      },
    },
    componentsOrder: ['region', 'endpointURL'],
  },
  credentials: {
    AccessKey: {
      components: {
        AWS_ACCESS_KEY_ID: {
          path: 'AWS_ACCESS_KEY_ID',
          uiType: 'text',
          fieldParams: { label: 'Access key ID' },
          validation: { required: true },
        },
        AWS_SECRET_ACCESS_KEY: {
          path: 'AWS_SECRET_ACCESS_KEY',
          uiType: 'text',
          fieldParams: { label: 'Secret access key', type: 'password' },
          validation: { required: true },
        },
      },
      componentsOrder: ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'],
    },
  },
});

// A second class whose AccessKey mode takes other keys, with the default
// location form and no parameters.
const ceph: BackupStorageClass = {
  metadata: { name: 'ceph' },
  spec: {
    displayName: 'Ceph RGW',
    authModes: [{ name: 'AccessKey', credentials: { required: ['CEPH_KEY'] } }],
  },
};
Reflect.set(ceph.spec, 'uiSchema', {
  credentials: {
    AccessKey: {
      components: {
        CEPH_KEY: {
          path: 'CEPH_KEY',
          uiType: 'text',
          fieldParams: { label: 'Ceph key' },
          validation: { required: true },
        },
      },
    },
  },
});

const conflict = new AxiosError('changed', '409', undefined, undefined, {
  status: 409,
  statusText: 'Conflict',
  data: {},
  headers: {},
  config: { headers: new AxiosHeaders() },
});

const S3_CREDENTIALS_LABELS = {
  'openeverest.io/backup-storage-class': 's3',
  'openeverest.io/definition': 'AccessKey',
};

const storedS3 = (): BackupStorageCRD => {
  const stored: BackupStorageCRD = {
    apiVersion: 'backup.openeverest.io/v1alpha1',
    kind: 'BackupStorage',
    metadata: { name: 's3-prod', namespace: 'team-a', resourceVersion: '5' },
    spec: {
      backupStorageClassRef: { name: 's3' },
      authentication: {
        mode: 'AccessKey',
        credentialsSecretRef: { name: 's3-prod-credentials' },
      },
      location: { objectStore: { bucket: 'backups', prefix: 'prod' } },
    },
  };
  Reflect.set(stored.spec, 'parameters', {
    region: 'us-east-1',
    endpointURL: 'https://minio.example.com',
    forcePathStyle: true,
  });
  return stored;
};

const PROD_TARGET: BackupStorageTarget = {
  namespace: 'team-a',
  name: 's3-prod',
};

const renderModal = (target?: BackupStorageTarget) => {
  const onClose = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <TestWrapper>
        <BackupStorageModal
          target={target}
          namespace="team-a"
          onClose={onClose}
        />
      </TestWrapper>
    </QueryClientProvider>
  );
  return onClose;
};

const fillS3Storage = async () => {
  fireEvent.change(await screen.findByTestId('text-input-name'), {
    target: { value: 's3-prod' },
  });
  fireEvent.mouseDown(
    within(screen.getByTestId('select-class-name-button')).getByRole('combobox')
  );
  fireEvent.click(
    screen.getByRole('option', { name: 'S3-compatible object storage' })
  );
  fireEvent.change(await screen.findByLabelText(/Bucket/), {
    target: { value: 'backups' },
  });
  fireEvent.change(screen.getByLabelText(/Prefix/), {
    target: { value: 'prod' },
  });
  fireEvent.change(screen.getByLabelText(/Region/), {
    target: { value: 'us-east-1' },
  });
  fireEvent.change(screen.getByLabelText(/Endpoint URL/), {
    target: { value: 'https://s3.us-east-1.amazonaws.com' },
  });
  fireEvent.change(await screen.findByLabelText(/Access key ID/), {
    target: { value: 'AKIA' },
  });
  fireEvent.change(screen.getByLabelText(/Secret access key/), {
    target: { value: 's3cr3t' },
  });
};

const changeRegionAndSave = async () => {
  fireEvent.change(await screen.findByLabelText(/Region/), {
    target: { value: 'eu-west-1' },
  });
  const save = screen.getByTestId('form-dialog-save');
  await waitFor(() => expect(save).toBeEnabled());
  fireEvent.click(save);
};

describe('BackupStorageModal', () => {
  beforeEach(() => {
    rbacMocks.check.mockReset().mockResolvedValue(true);
    vi.mocked(getBackupStorageClassesFn).mockReset().mockResolvedValue([s3]);
    vi.mocked(getBackupStorageFn).mockReset().mockResolvedValue(storedS3());
    vi.mocked(createSecretFn).mockReset().mockResolvedValue({});
    vi.mocked(getSecretFn).mockReset();
    vi.mocked(patchSecretFn).mockReset();
    vi.mocked(createBackupStorageFn).mockReset().mockResolvedValue(storedS3());
    vi.mocked(updateBackupStorageFn).mockReset().mockResolvedValue(storedS3());
  });

  // Filling the whole add form is slow under a parallel run.
  it(
    'adds a storage at its location, its credentials in a Secret of its class',
    {
      timeout: 15000,
    },
    async () => {
      const onClose = renderModal();
      await fillS3Storage();
      const add = screen.getByTestId('form-dialog-add');
      await waitFor(() => expect(add).toBeEnabled());
      fireEvent.click(add);

      await waitFor(() => expect(onClose).toHaveBeenCalled());
      expect(createSecretFn).toHaveBeenCalledWith('main', 'team-a', {
        metadata: {
          name: 'backup-storage-s3-prod-credentials',
          labels: S3_CREDENTIALS_LABELS,
        },
        stringData: {
          AWS_ACCESS_KEY_ID: 'AKIA',
          AWS_SECRET_ACCESS_KEY: 's3cr3t',
        },
      });
      expect(createBackupStorageFn).toHaveBeenCalledWith('main', 'team-a', {
        metadata: { name: 's3-prod' },
        spec: {
          backupStorageClassRef: { name: 's3' },
          authentication: {
            mode: 'AccessKey',
            credentialsSecretRef: {
              name: 'backup-storage-s3-prod-credentials',
            },
          },
          location: { objectStore: { bucket: 'backups', prefix: 'prod' } },
          parameters: {
            region: 'us-east-1',
            endpointURL: 'https://s3.us-east-1.amazonaws.com',
          },
        },
      });
    }
  );

  it(
    "doesn't put the credentials typed for another type into the Secret",
    {
      timeout: 15000,
    },
    async () => {
      vi.mocked(getBackupStorageClassesFn).mockResolvedValue([s3, ceph]);
      const onClose = renderModal();
      await fillS3Storage();

      fireEvent.mouseDown(
        within(screen.getByTestId('select-class-name-button')).getByRole(
          'combobox'
        )
      );
      fireEvent.click(screen.getByRole('option', { name: 'Ceph RGW' }));
      fireEvent.change(await screen.findByLabelText(/Bucket/), {
        target: { value: 'backups' },
      });
      fireEvent.change(await screen.findByLabelText(/Ceph key/), {
        target: { value: 'ck' },
      });
      const add = screen.getByTestId('form-dialog-add');
      await waitFor(() => expect(add).toBeEnabled());
      fireEvent.click(add);

      await waitFor(() => expect(onClose).toHaveBeenCalled());
      expect(createSecretFn).toHaveBeenCalledWith('main', 'team-a', {
        metadata: {
          name: 'backup-storage-s3-prod-credentials',
          labels: {
            'openeverest.io/backup-storage-class': 'ceph',
            'openeverest.io/definition': 'AccessKey',
          },
        },
        stringData: { CEPH_KEY: 'ck' },
      });
    }
  );

  it('keeps the location and the parameters its form lacks on an edit', async () => {
    const onClose = renderModal(PROD_TARGET);

    expect(await screen.findByLabelText(/Bucket/)).toBeDisabled();
    expect(screen.getByLabelText(/Prefix/)).toBeDisabled();
    await changeRegionAndSave();

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(rbacMocks.check).toHaveBeenCalledWith(
      'use',
      'secrets',
      'team-a/s3-prod-credentials'
    );
    expect(createSecretFn).not.toHaveBeenCalled();
    expect(patchSecretFn).not.toHaveBeenCalled();
    expect(updateBackupStorageFn).toHaveBeenCalledWith('main', 'team-a', {
      ...storedS3(),
      spec: {
        ...storedS3().spec,
        parameters: {
          region: 'eu-west-1',
          endpointURL: 'https://minio.example.com',
          forcePathStyle: true,
        },
      },
    });
  });

  it("saves a storage whose name the add form wouldn't take, as the API took it", async () => {
    const stored = storedS3();
    stored.metadata = { ...stored.metadata, name: 's3.prod' };
    vi.mocked(getBackupStorageFn).mockResolvedValue(stored);
    const onClose = renderModal({ namespace: 'team-a', name: 's3.prod' });

    await changeRegionAndSave();

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(updateBackupStorageFn).toHaveBeenCalledWith(
      'main',
      'team-a',
      expect.objectContaining({
        metadata: expect.objectContaining({ name: 's3.prod' }),
      })
    );
  });

  it('refuses changing the storage without permission to use its Secret, writing nothing', async () => {
    rbacMocks.check.mockResolvedValue(false);
    const onClose = renderModal(PROD_TARGET);

    await changeRegionAndSave();

    expect(
      await screen.findByText(
        /needs permission to use Secret s3-prod-credentials in the team-a namespace/
      )
    ).toBeInTheDocument();
    expect(updateBackupStorageFn).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('saves over the latest version, the user edits kept, once the storage changed meanwhile', async () => {
    const changed = storedS3();
    changed.metadata = { ...changed.metadata, resourceVersion: '6' };
    changed.spec.tls = { insecureSkipVerify: true };
    vi.mocked(getBackupStorageFn)
      .mockResolvedValueOnce(storedS3())
      .mockResolvedValue(changed);
    vi.mocked(updateBackupStorageFn).mockRejectedValueOnce(conflict);
    const onClose = renderModal(PROD_TARGET);

    await changeRegionAndSave();

    expect(
      await screen.findByText(/This storage was changed meanwhile/)
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByLabelText('Skip TLS certificate verification')
      ).toBeChecked()
    );
    expect(screen.getByLabelText(/Region/)).toHaveValue('eu-west-1');
    fireEvent.click(screen.getByTestId('form-dialog-save'));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(updateBackupStorageFn).toHaveBeenLastCalledWith(
      'main',
      'team-a',
      expect.objectContaining({
        metadata: expect.objectContaining({ resourceVersion: '6' }),
        spec: expect.objectContaining({
          tls: { insecureSkipVerify: true },
          parameters: expect.objectContaining({ region: 'eu-west-1' }),
        }),
      })
    );
  });

  it("can't save a storage whose class is gone, which would drop its parameters", async () => {
    vi.mocked(getBackupStorageClassesFn).mockResolvedValue([]);
    renderModal(PROD_TARGET);

    expect(
      await screen.findByText(/This storage's type, s3, is no longer available/)
    ).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('Skip TLS certificate verification'));
    expect(screen.getByTestId('form-dialog-save')).toBeDisabled();
    expect(updateBackupStorageFn).not.toHaveBeenCalled();
  });
});
