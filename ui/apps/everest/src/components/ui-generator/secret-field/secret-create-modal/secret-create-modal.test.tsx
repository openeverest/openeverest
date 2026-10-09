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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Provider } from 'shared-types/api.types';
import { TestWrapper } from 'utils/test';
import { FieldType, Section } from '../../ui-generator.types';
import { SecretCreateModal } from './secret-create-modal';
import { Messages } from './secret-create-modal.messages';

const mockMutate = vi.fn();

vi.mock('hooks/api/secrets', () => {
  const createSecret = {
    mutate: (...args: unknown[]) => mockMutate(...args),
    isPending: false,
  };
  return { useCreateSecret: () => createSecret };
});

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));

const PROVIDER: Provider = { metadata: { name: 'psmdb' }, spec: {} };

// Paths are the secret's data keys.
const SECTION: Section = {
  label: 'Database user',
  components: {
    user: {
      uiType: FieldType.Text,
      path: 'MONGODB_USER',
      fieldParams: { label: 'User' },
      validation: { required: true },
    },
    port: {
      uiType: FieldType.Number,
      path: 'PMM_PORT',
      fieldParams: { label: 'Port', defaultValue: 443 },
    },
  },
};

const renderModal = () => {
  const onCreated = vi.fn();
  render(
    <TestWrapper>
      <SecretCreateModal
        definition="user"
        section={SECTION}
        providerObject={PROVIDER}
        namespace="ns"
        onClose={vi.fn()}
        onCreated={onCreated}
      />
    </TestWrapper>
  );
  return onCreated;
};

const getNameInput = () =>
  screen.getByLabelText(new RegExp(Messages.nameLabel));
const getCreateButton = () =>
  screen.getByRole('button', { name: Messages.create });

const fillAndCreate = async () => {
  fireEvent.change(screen.getByLabelText(/User/), {
    target: { value: 'admin' },
  });
  await waitFor(() => expect(getCreateButton()).toBeEnabled());
  fireEvent.click(getCreateButton());
};

describe('SecretCreateModal', () => {
  beforeEach(() => {
    mockMutate.mockReset();
  });

  it('creates the secret from the form data and reports its name', async () => {
    mockMutate.mockImplementation((_secret, { onSuccess }) => onSuccess());
    const onCreated = renderModal();

    await fillAndCreate();

    await waitFor(() => expect(onCreated).toHaveBeenCalled());
    const [secret] = mockMutate.mock.calls[0];
    expect(secret).toEqual({
      metadata: {
        name: expect.stringMatching(/^user-[a-z0-9]{3}$/),
        labels: {
          'openeverest.io/provider': 'psmdb',
          'openeverest.io/definition': 'user',
        },
      },
      stringData: { MONGODB_USER: 'admin', PMM_PORT: '443' },
    });
    expect(getNameInput()).toHaveValue(secret.metadata.name);
    expect(onCreated).toHaveBeenCalledWith(secret.metadata.name);
  });

  it('flags a name the server reports as taken until it changes', async () => {
    mockMutate.mockImplementation((_secret, { onError }) =>
      onError({ status: 409 })
    );
    const onCreated = renderModal();

    await fillAndCreate();

    expect(await screen.findByText(Messages.nameTaken)).toBeInTheDocument();
    expect(getCreateButton()).toBeDisabled();
    expect(onCreated).not.toHaveBeenCalled();

    fireEvent.change(getNameInput(), { target: { value: 'user-other' } });

    await waitFor(() =>
      expect(screen.queryByText(Messages.nameTaken)).not.toBeInTheDocument()
    );
    expect(getCreateButton()).toBeEnabled();
  });
});
