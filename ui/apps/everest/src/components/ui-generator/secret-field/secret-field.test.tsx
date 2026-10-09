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
import { FormProvider, useForm } from 'react-hook-form';
import { Provider } from 'shared-types/api.types';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import {
  FieldType,
  FormMode,
  SecretFieldParams,
  TopologyUISchemas,
} from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { Messages } from './secret-field.messages';

const mockUseSecrets = vi.fn();
const mockMutate = vi.fn();
const mockPermissions = { canCreate: true };

vi.mock('hooks/api/secrets', () => {
  const createSecret = {
    mutate: (...args: unknown[]) => mockMutate(...args),
    isPending: false,
  };
  return {
    useSecrets: (...args: unknown[]) => mockUseSecrets(...args),
    useCreateSecret: () => createSecret,
  };
});

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => mockPermissions,
}));

const LOADED = {
  data: [{ metadata: { name: 'creds-a' } }, { metadata: { name: 'creds-b' } }],
  isLoading: false,
  isError: false,
  isSuccess: true,
};

// What a disabled query returns.
const IDLE = { isLoading: false, isError: false, isSuccess: false };

const PROVIDER: Provider = { metadata: { name: 'psmdb' }, spec: {} };

// uiSchema is opaque (Record<string, never>) in the generated Provider type.
const PROVIDER_WITH_FORM = {
  metadata: { name: 'psmdb' },
  spec: {
    secrets: {
      userSecret: {
        uiSchema: {
          label: 'Database user',
          components: {
            user: {
              uiType: FieldType.Text,
              path: 'USER',
              fieldParams: { label: 'User' },
              validation: { required: true },
            },
          },
        },
      },
    },
  },
} as unknown as Provider;

const buildSchema = (
  fieldParams: Partial<SecretFieldParams> = {}
): TopologyUISchemas => ({
  replicaSet: {
    sections: {
      basicInfo: {
        components: {
          userSecret: {
            uiType: FieldType.Secret,
            path: 'spec.userSecretRef.name',
            fieldParams: {
              label: 'Credentials',
              definition: 'userSecret',
              ...fieldParams,
            },
            validation: { required: true },
          },
        },
      },
    },
  },
});

const schema = buildSchema();

interface FieldContext {
  providerObject?: Provider;
  formMode?: FormMode;
  fieldParams?: Partial<SecretFieldParams>;
  value?: string;
}

const renderField = ({ fieldParams, value, ...context }: FieldContext) => {
  const fieldSchema = buildSchema(fieldParams);
  const Harness = () => {
    const methods = useForm({
      defaultValues: value
        ? { spec: { userSecretRef: { name: value } } }
        : getDefaultValues(fieldSchema, 'replicaSet'),
    });
    return (
      <FormProvider {...methods}>
        <UIGenerator
          sections={fieldSchema.replicaSet!.sections}
          sectionKey="basicInfo"
          namespace="ns"
          {...context}
        />
      </FormProvider>
    );
  };

  render(
    <TestWrapper>
      <Harness />
    </TestWrapper>
  );
};

const getCombobox = () =>
  within(
    screen.getByTestId('select-spec.user-secret-ref.name-button')
  ).getByRole('combobox');
const queryAddButton = () => screen.queryByRole('button', { name: 'Add new' });

describe('SecretField', () => {
  beforeEach(() => {
    mockUseSecrets.mockReset();
    mockUseSecrets.mockReturnValue(LOADED);
    mockMutate.mockReset();
    mockPermissions.canCreate = true;
  });

  it('creates a secret from its definition form and selects it', async () => {
    mockMutate.mockImplementation((secret, { onSuccess }) => {
      mockUseSecrets.mockReturnValue({
        ...LOADED,
        data: [...LOADED.data, { metadata: secret.metadata }],
      });
      onSuccess();
    });
    renderField({ providerObject: PROVIDER_WITH_FORM });

    fireEvent.click(screen.getByRole('button', { name: 'Add new' }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.change(within(dialog).getByLabelText(/User/), {
      target: { value: 'admin' },
    });
    const create = within(dialog).getByRole('button', { name: 'Create' });
    await waitFor(() => expect(create).toBeEnabled());
    fireEvent.click(create);

    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    );
    const [secret] = mockMutate.mock.calls[0];
    expect(secret.stringData).toEqual({ USER: 'admin' });
    expect(getCombobox()).toHaveTextContent(secret.metadata.name);
  });

  it.each<[string, FieldContext & { canCreate?: boolean }]>([
    [
      'without create permission',
      { canCreate: false, providerObject: PROVIDER_WITH_FORM },
    ],
    [
      'in edit mode',
      { providerObject: PROVIDER_WITH_FORM, formMode: FormMode.Edit },
    ],
    [
      'for a read-only field',
      { providerObject: PROVIDER_WITH_FORM, fieldParams: { readOnly: true } },
    ],
    ['when the definition has no create form', { providerObject: PROVIDER }],
  ])('offers no Add new %s', (_, { canCreate = true, ...field }) => {
    mockPermissions.canCreate = canCreate;
    renderField(field);

    expect(getCombobox()).toBeInTheDocument();
    expect(queryAddButton()).not.toBeInTheDocument();
  });

  it("offers the provider's secrets of the field's definition", () => {
    renderField({ providerObject: PROVIDER });

    expect(mockUseSecrets).toHaveBeenCalledWith(
      'main',
      'ns',
      { provider: 'psmdb', definition: 'userSecret' },
      { enabled: true }
    );

    fireEvent.mouseDown(getCombobox());

    expect(screen.getByRole('option', { name: 'creds-a' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'creds-b' })).toBeInTheDocument();
  });

  it('is disabled without querying when there is no provider', () => {
    renderField({});

    expect(mockUseSecrets).toHaveBeenCalledWith(
      'main',
      'ns',
      { provider: undefined, definition: 'userSecret' },
      { enabled: false }
    );
    expect(getCombobox()).toHaveAttribute('aria-disabled', 'true');
    expect(screen.getByText(Messages.noContext)).toBeInTheDocument();
  });

  it('keeps a value that is not a managed secret, marked as such', () => {
    renderField({ providerObject: PROVIDER, value: 'legacy-creds' });

    expect(getCombobox()).toHaveTextContent(Messages.unmanaged('legacy-creds'));
  });

  it('shows the value of a read-only field without listing secrets', () => {
    mockUseSecrets.mockReturnValue(IDLE);
    renderField({
      providerObject: PROVIDER,
      fieldParams: { readOnly: true },
      value: 'creds-a',
    });

    expect(mockUseSecrets).toHaveBeenCalledWith(
      'main',
      'ns',
      { provider: 'psmdb', definition: 'userSecret' },
      { enabled: false }
    );
    expect(getCombobox()).toHaveTextContent(/^creds-a$/);
    expect(screen.queryByText(Messages.empty)).not.toBeInTheDocument();
  });

  it('requires a secret name when the field is required', () => {
    const { schema: zodSchema } = buildZodSchema(schema, 'replicaSet');
    const withName = (name: string) => ({
      spec: { userSecretRef: { name } },
    });

    expect(zodSchema.safeParse(withName('')).success).toBe(false);
    expect(zodSchema.safeParse(withName('creds-a')).success).toBe(true);
  });
});
