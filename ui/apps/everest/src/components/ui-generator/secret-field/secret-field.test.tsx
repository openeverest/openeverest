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

import { fireEvent, render, screen, within } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { Provider } from 'shared-types/api.types';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { FieldType, TopologyUISchemas } from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { Messages } from './secret-field.messages';

const mockUseSecrets = vi.fn();

vi.mock('hooks/api/secrets', () => ({
  useSecrets: (...args: unknown[]) => mockUseSecrets(...args),
}));

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => 'main',
}));

const LOADED = {
  data: [{ metadata: { name: 'creds-a' } }, { metadata: { name: 'creds-b' } }],
  isLoading: false,
  isError: false,
};

const PROVIDER: Provider = { metadata: { name: 'psmdb' }, spec: {} };

const schema: TopologyUISchemas = {
  replicaSet: {
    sections: {
      basicInfo: {
        components: {
          userSecret: {
            uiType: FieldType.Secret,
            path: 'spec.userSecretRef.name',
            fieldParams: { label: 'Credentials', definition: 'userSecret' },
            validation: { required: true },
          },
        },
      },
    },
  },
};

const renderField = (context: { providerObject?: Provider }) => {
  const Harness = () => {
    const methods = useForm({
      defaultValues: getDefaultValues(schema, 'replicaSet'),
    });
    return (
      <FormProvider {...methods}>
        <UIGenerator
          sections={schema.replicaSet!.sections}
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

describe('SecretField', () => {
  beforeEach(() => {
    mockUseSecrets.mockReset();
    mockUseSecrets.mockReturnValue(LOADED);
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

  it('requires a secret name when the field is required', () => {
    const { schema: zodSchema } = buildZodSchema(schema, 'replicaSet');
    const withName = (name: string) => ({
      spec: { userSecretRef: { name } },
    });

    expect(zodSchema.safeParse(withName('')).success).toBe(false);
    expect(zodSchema.safeParse(withName('creds-a')).success).toBe(true);
  });
});
