import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { BackupStorageSecretConflictError } from 'api/backupStorage';
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
      spec: { ...s3Class.spec, authModes: [s3Class.spec.authModes[0]] },
    },
  ];
  return {
    useBackupStorageClasses: () => ({ data: classes, isLoading: false }),
  };
});
vi.mock('hooks/api/secrets', () => ({
  useSecrets: () => ({
    data: [{ metadata: { name: 'team-creds' } }],
    isFetching: false,
  }),
}));

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
          location: { bucket: 'my-backups' },
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
      expect(
        screen.getByText(/Leave the fields blank to keep the current Secret/)
      ).toBeInTheDocument();

      const submit = screen.getByTestId('form-dialog-edit');
      await waitFor(() => expect(submit).toBeEnabled());
      fireEvent.click(submit);

      await waitFor(() => expect(handleSubmitModal).toHaveBeenCalled());
      expect(handleSubmitModal.mock.calls[0]).toEqual([
        true,
        expect.objectContaining({
          credentialsSource: CredentialsSource.Existing,
          credentialsSecretName: 'backup-storage-s3-prod-credentials',
        }),
      ]);
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
      fireEvent.click(screen.getByRole('button', { name: 'Use that Secret' }));

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
});
