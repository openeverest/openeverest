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
import { BackupStorageSecretConflictError } from 'api/backupStorage';
import { Secret } from 'shared-types/api.types';
import {
  BackupStorageSecretKind,
  CredentialsSource,
} from 'shared-types/backupStorages.types';
import { TestWrapper } from 'utils/test';
import { validateInputWithRFC1035 } from 'utils/tests/validate-rfc1035';
import { s3Storage } from '../storage-locations.fixtures';
import { CreateEditStorageForm } from './create-edit-form';
import { CreateEditModalStorage } from './create-edit-modal';
import { CreateEditModalStorageProps } from './create-edit-modal.types';

const mocks = vi.hoisted(() => {
  type SecretQuery = { data?: Secret; isError: boolean };
  const managedSecret: SecretQuery = {
    data: {
      metadata: {
        name: 'backup-storage-s3-prod-credentials',
        labels: { 'openeverest.io/managed': 'true' },
      },
    },
    isError: false,
  };
  const missingSecret: SecretQuery = { isError: true };
  return { managedSecret, missingSecret, secret: { current: managedSecret } };
});

vi.mock('hooks/api/namespaces/useNamespaces', () => ({
  useNamespaces: () => ({
    data: ['team-a'],
    isFetching: false,
  }),
}));
vi.mock('hooks/api/backup-storage-classes', async () => {
  const { s3Class } = await import('../storage-locations.fixtures');
  const classes = [
    {
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
    },
  ];
  return {
    useBackupStorageClasses: () => ({ data: classes, isLoading: false }),
  };
});
vi.mock('hooks/api/secrets', () => {
  const caSecrets = {
    data: [{ metadata: { name: 'corp-ca' } }],
    isFetching: false,
  };
  const credentialsSecrets = {
    data: [
      { metadata: { name: 'team-creds' } },
      { metadata: { name: 'backup-storage-s3-prod-credentials' } },
    ],
    isFetching: false,
  };
  return {
    // The credentials picker lists Secrets of the class and mode; the CA
    // picker lists ca-bundle Secrets.
    useSecrets: (
      _cluster: string,
      _namespace: string,
      filter: { definition?: string }
    ) => (filter.definition === 'ca-bundle' ? caSecrets : credentialsSecrets),
    useSecret: () => mocks.secret.current,
  };
});
vi.mock('hooks/rbac', async (importOriginal) => {
  const permissions = {
    canRead: true,
    canUpdate: true,
    canCreate: true,
    canDelete: true,
    isPermissionsLoading: false,
  };
  return {
    ...(await importOriginal<typeof import('hooks/rbac')>()),
    useRBACPermissions: () => permissions,
  };
});

beforeEach(() => {
  mocks.secret.current = mocks.managedSecret;
});

const errors = {
  MIN1_ERROR: 'String must contain at least 1 character(s)',
  MAX22_ERROR: 'String must contain at most 22 character(s)',
  SPECIAL_CHAR_ERROR:
    'The storage name should only contain lowercase letters, numbers and hyphens.',
  START_CHAR_ERROR: "The name shouldn't start with a hyphen or a number.",
  END_CHAR_ERROR: "The name shouldn't end with a hyphen.",
};

validateInputWithRFC1035({
  renderComponent: () => <CreateEditStorageForm isEditMode={false} />,
  suiteName: 'Backup storage modal',
  errors: errors,
});

const renderModal = (props: Partial<CreateEditModalStorageProps> = {}) => {
  const handleSubmitModal = vi.fn();
  render(
    <TestWrapper>
      <CreateEditModalStorage
        open
        handleCloseModal={vi.fn()}
        handleSubmitModal={handleSubmitModal}
        prefillNamespace="team-a"
        {...props}
      />
    </TestWrapper>
  );
  return handleSubmitModal;
};

const textbox = (name: RegExp) => screen.getByRole('textbox', { name });
const textboxOrNull = (name: RegExp) => screen.queryByRole('textbox', { name });

const type = (element: HTMLElement, value: string) =>
  fireEvent.change(element, { target: { value } });

describe('CreateEditModalStorage', () => {
  it('renders the fields of the only class', { timeout: 30000 }, async () => {
    renderModal();

    await waitFor(() =>
      expect(
        screen.getByTestId('select-input-backup-storage-class')
      ).toHaveValue('s3')
    );
    expect(
      screen.getByText('Amazon S3 and S3-compatible object stores')
    ).toBeInTheDocument();
    [
      /^Bucket/,
      /^Prefix/,
      /^Region/,
      /^Endpoint URL/,
      /^Access key ID/,
    ].forEach((name) => expect(textbox(name)).toBeEnabled());
    expect(screen.getByLabelText(/^Secret access key/)).toHaveAttribute(
      'type',
      'password'
    );
    expect(
      screen.getByRole('combobox', { name: /^Implementation/ })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('switch', { name: 'Force path-style URLs' })
    ).not.toBeChecked();
  });

  it('submits the values the class renders', { timeout: 30000 }, async () => {
    const handleSubmitModal = renderModal();

    type(textbox(/^Name/), 's3-prod');
    type(await waitFor(() => textbox(/^Bucket/)), 'my-backups');
    type(textbox(/^Region/), 'us-east-1');
    type(textbox(/^Endpoint URL/), 'https://minio.local');
    type(textbox(/^Access key ID/), 'id');
    type(screen.getByLabelText(/^Secret access key/), 'key');

    const submit = screen.getByTestId('form-dialog-add');
    await waitFor(() => expect(submit).toBeEnabled());
    fireEvent.click(submit);

    await waitFor(() => expect(handleSubmitModal).toHaveBeenCalled());
    expect(handleSubmitModal.mock.calls[0]).toEqual([
      false,
      expect.objectContaining({
        name: 's3-prod',
        namespace: 'team-a',
        backupStorageClass: 's3',
        spec: {
          location: { objectStore: { bucket: 'my-backups' } },
          parameters: {
            region: 'us-east-1',
            endpointURL: 'https://minio.local',
            forcePathStyle: false,
          },
        },
        authMode: 'AccessKey',
        credentialsSource: CredentialsSource.New,
        credentials: { AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'key' },
      }),
    ]);
  });

  it(
    'keeps the class and location read-only on edit',
    { timeout: 30000 },
    async () => {
      const handleSubmitModal = renderModal({
        selectedStorageLocation: s3Storage(),
      });

      expect(
        screen.getByTestId('select-input-backup-storage-class')
      ).toBeDisabled();
      expect(textbox(/^Bucket/)).toBeDisabled();
      expect(textbox(/^Bucket/)).toHaveValue('my-backups');
      expect(textbox(/^Prefix/)).toBeDisabled();
      expect(textbox(/^Region/)).toBeEnabled();
      expect(textbox(/^Region/)).toHaveValue('us-east-1');

      const submit = screen.getByTestId('form-dialog-edit');
      await waitFor(() => expect(submit).toBeEnabled());
      fireEvent.click(submit);

      await waitFor(() => expect(handleSubmitModal).toHaveBeenCalled());
      expect(handleSubmitModal.mock.calls[0]).toEqual([
        true,
        expect.objectContaining({
          credentialsSource: CredentialsSource.Current,
          credentialsSecretName: 'backup-storage-s3-prod-credentials',
          credentials: {},
          removedCredentialKeys: [],
        }),
      ]);
    }
  );

  it(
    'replaces and removes single keys of the current credentials',
    { timeout: 30000 },
    async () => {
      const handleSubmitModal = renderModal({
        selectedStorageLocation: s3Storage(),
      });

      const secretKeyRow = screen.getByTestId(
        'credentials-key-AWS_SECRET_ACCESS_KEY'
      );
      expect(secretKeyRow).toHaveTextContent('Configured');
      expect(
        screen.getByTestId('credentials-key-AWS_ACCESS_KEY_ID')
      ).toHaveTextContent('Configured');
      expect(
        screen.queryByLabelText(/^Secret access key/)
      ).not.toBeInTheDocument();
      expect(
        screen.getByText('Optional · leave blank to keep the current value')
      ).toBeInTheDocument();

      fireEvent.click(
        within(secretKeyRow).getByRole('button', { name: 'Replace' })
      );
      type(screen.getByLabelText(/^Secret access key/), 'rotated');
      fireEvent.click(screen.getByRole('button', { name: 'Remove' }));
      expect(
        screen.getByTestId('credentials-key-AWS_SESSION_TOKEN')
      ).toHaveTextContent('Removed when you save');

      const submit = screen.getByTestId('form-dialog-edit');
      await waitFor(() => expect(submit).toBeEnabled());
      fireEvent.click(submit);

      await waitFor(() => expect(handleSubmitModal).toHaveBeenCalled());
      expect(handleSubmitModal.mock.calls[0]).toEqual([
        true,
        expect.objectContaining({
          credentialsSource: CredentialsSource.Current,
          credentialsSecretName: 'backup-storage-s3-prod-credentials',
          credentials: { AWS_SECRET_ACCESS_KEY: 'rotated' },
          removedCredentialKeys: ['AWS_SESSION_TOKEN'],
        }),
      ]);
    }
  );

  it(
    "explains why the current credentials can't be changed",
    { timeout: 30000 },
    async () => {
      mocks.secret.current = mocks.missingSecret;
      renderModal({ selectedStorageLocation: s3Storage() });

      expect(
        screen.getByText(
          /Credentials backup-storage-s3-prod-credentials can't be read/
        )
      ).toBeInTheDocument();
      screen
        .getAllByRole('button', { name: 'Replace' })
        .forEach((button) => expect(button).toBeDisabled());
      expect(screen.getByRole('button', { name: 'Remove' })).toBeDisabled();

      fireEvent.click(screen.getByLabelText('Use other saved credentials'));
      expect(
        await screen.findByTestId('select-input-credentials-secret-name')
      ).toHaveValue('backup-storage-s3-prod-credentials');
    }
  );

  it(
    'asks for the key only for certificates from another tool',
    { timeout: 30000 },
    async () => {
      renderModal({
        selectedStorageLocation: s3Storage({
          tls: { ca: { secretKeyRef: { name: 'corp-ca', key: 'ca.crt' } } },
        }),
      });

      expect(screen.getByLabelText('Use saved certificates')).toBeChecked();
      expect(screen.getByTestId('select-input-ca-secret-name')).toHaveValue(
        'corp-ca'
      );
      expect(textboxOrNull(/^Key in the Secret/)).not.toBeInTheDocument();

      fireEvent.click(
        screen.getByLabelText(
          'Certificates from another tool, such as cert-manager'
        )
      );
      expect(textbox(/^Key in the Secret/)).toHaveValue('ca.crt');
    }
  );

  it(
    'offers the Secret that blocked the credentials',
    { timeout: 30000 },
    async () => {
      renderModal({
        secretConflict: new BackupStorageSecretConflictError(
          BackupStorageSecretKind.Credentials,
          'backup-storage-s3-prod-credentials'
        ),
      });

      expect(
        await screen.findByTestId('secret-conflict-alert')
      ).toHaveTextContent('backup-storage-s3-prod-credentials');
      fireEvent.click(screen.getByRole('button', { name: 'Use them' }));

      await waitFor(() =>
        expect(
          screen.queryByTestId('secret-conflict-alert')
        ).not.toBeInTheDocument()
      );
      expect(
        screen.getByTestId('select-input-credentials-secret-name')
      ).toHaveValue('backup-storage-s3-prod-credentials');
    }
  );

  it(
    'does not offer a taken name whose Secret holds other credentials',
    { timeout: 30000 },
    async () => {
      renderModal({
        secretConflict: new BackupStorageSecretConflictError(
          BackupStorageSecretKind.Credentials,
          'backup-storage-s3-staging-credentials'
        ),
      });

      expect(
        await screen.findByTestId('secret-conflict-alert')
      ).toHaveTextContent(
        "doesn't hold credentials for this storage type and authentication mode"
      );
      expect(
        screen.queryByRole('button', { name: 'Use them' })
      ).not.toBeInTheDocument();
    }
  );
});
