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

import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import rbacModel from '/src/static/model.conf?raw';
import type { StorageClass } from 'shared-types/kubernetes.types';
import {
  initializeAuthorizerFetchLoop,
  stopAuthorizerFetchLoop,
} from 'utils/rbac';
import { TestWrapper } from 'utils/test';
import { UiGeneratorProvider } from '../ui-generator-context';
import { type Component, FieldType, FormMode } from '../ui-generator.types';
import {
  isSelectComponent,
  resolveSelectOptions,
} from '../ui-component/utils/select-component-handler';
import { DataSourceField } from './data-source-field';
import type { ComponentWithDataSource } from './data-source-field';
import { Messages as DataSourceMessages } from './data-source-field/data-source-field.messages';
import { Messages } from './providers.messages';
import './providers';

// Only the network is mocked: the storageClasses provider, its query, the
// default-class sort, the RBAC hook and the Casbin authorizer all run for real.
const { mockGetStorageClasses, mockGetRBACPolicies } = vi.hoisted(() => ({
  mockGetStorageClasses: vi.fn(),
  mockGetRBACPolicies: vi.fn(),
}));

vi.mock('api/storageClassesApi', () => ({
  getStorageClassesFn: (cluster: string) => mockGetStorageClasses(cluster),
}));

vi.mock('api/policies', () => ({
  getRBACPolicies: () => mockGetRBACPolicies(),
}));

const USER = 'dev';
const FIELD = 'spec.storageClass';

const ITEM: ComponentWithDataSource = {
  uiType: FieldType.Select,
  path: FIELD,
  fieldParams: { label: 'Storage class' },
  dataSource: { provider: 'storageClasses' },
};

const storageClass = (name: string, isDefault = false): StorageClass => ({
  metadata: {
    name,
    ...(isDefault
      ? {
          annotations: {
            'storageclass.kubernetes.io/is-default-class': 'true',
          },
        }
      : {}),
  },
  provisioner: 'rancher.io/local-path',
});

const givenPolicies = async (enabled: boolean, policies: string[][] = []) => {
  mockGetRBACPolicies.mockResolvedValue({
    enabled,
    m: rbacModel,
    p: policies.map((policy) => ['p', USER, ...policy]),
  });
  await initializeAuthorizerFetchLoop(USER);
};

const SelectedValue = () => (
  <output data-testid="value">{useWatch({ name: FIELD })}</output>
);

const renderField = (patched: Component) =>
  isSelectComponent(patched) ? (
    <>
      <ul>
        {resolveSelectOptions(patched.fieldParams).map((option) => (
          <li key={option.value}>{option.label}</li>
        ))}
      </ul>
      <p data-testid="helper-text">{patched.fieldParams.helperText}</p>
    </>
  ) : null;

const renderStorageClassField = ({
  formMode = FormMode.New,
  storedStorageClass = '',
}: {
  formMode?: FormMode;
  storedStorageClass?: string;
} = {}) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  const Harness = () => {
    const methods = useForm({
      defaultValues: { spec: { storageClass: storedStorageClass } },
    });
    return (
      <FormProvider {...methods}>
        <DataSourceField item={ITEM} name={FIELD}>
          {renderField}
        </DataSourceField>
        <SelectedValue />
      </FormProvider>
    );
  };

  render(
    <TestWrapper>
      <QueryClientProvider client={queryClient}>
        <UiGeneratorProvider namespace="ns" formMode={formMode}>
          <Harness />
        </UiGeneratorProvider>
      </QueryClientProvider>
    </TestWrapper>
  );
};

describe('storageClasses data source', () => {
  beforeEach(() => {
    mockGetStorageClasses.mockReset();
    mockGetRBACPolicies.mockReset();
  });

  afterEach(() => {
    stopAuthorizerFetchLoop();
  });

  it('preselects the default class even when the API lists it last', async () => {
    await givenPolicies(false);
    mockGetStorageClasses.mockResolvedValue({
      items: [storageClass('fast-ssd'), storageClass('local-path', true)],
    });

    renderStorageClassField();

    await waitFor(() =>
      expect(screen.getByTestId('value')).toHaveTextContent('local-path')
    );
    expect(mockGetStorageClasses).toHaveBeenCalledWith('main');
    expect(
      screen.getAllByRole('listitem').map((item) => item.textContent)
    ).toEqual(['local-path', 'fast-ssd']);
  });

  it('explains an empty list when the role has no storage-classes read', async () => {
    await givenPolicies(true, [
      ['namespaces', '*', 'main/*'],
      ['instances', '*', 'main/*/*'],
    ]);
    mockGetStorageClasses.mockResolvedValue({ items: [] });

    renderStorageClassField();

    await waitFor(() =>
      expect(screen.getByTestId('helper-text')).toHaveTextContent(
        Messages.storageClassesNoReadPermission
      )
    );
  });

  // The role can't list classes, but an existing Instance keeps the class its
  // spec holds (or none), so the "default will be used" hint would be wrong.
  it.each([
    ['a stored class', 'standard', DataSourceMessages.unavailableValueHint],
    ['an unset class', '', Messages.storageClassesUnsetValueHint],
  ])(
    'explains %s in edit instead of the missing storage-classes read',
    async (_case, storedStorageClass, expectedHint) => {
      await givenPolicies(true, [['instances', '*', 'main/*/*']]);
      mockGetStorageClasses.mockResolvedValue({ items: [] });

      renderStorageClassField({
        formMode: FormMode.Edit,
        storedStorageClass,
      });

      await waitFor(() =>
        expect(screen.getByTestId('helper-text')).toHaveTextContent(
          expectedHint
        )
      );
    }
  );

  it('does not blame permissions when the role may read but the cluster has no classes', async () => {
    await givenPolicies(true, [['storage-classes', 'read', 'main/*']]);
    mockGetStorageClasses.mockResolvedValue({ items: [] });

    renderStorageClassField();

    await waitFor(() =>
      expect(screen.getByTestId('helper-text')).toHaveTextContent(
        DataSourceMessages.noOptions
      )
    );
  });

  // A grant on specific classes doesn't match the `main/*` object that a
  // whole-resource permission check uses. The classes the API returned for
  // that grant must still show, without a missing-permission hint.
  it('shows classes granted by name without a permission hint', async () => {
    await givenPolicies(true, [['storage-classes', 'read', 'main/standard']]);
    mockGetStorageClasses.mockResolvedValue({
      items: [storageClass('standard')],
    });

    renderStorageClassField();

    await waitFor(() =>
      expect(screen.getByTestId('value')).toHaveTextContent('standard')
    );
    expect(screen.getByTestId('helper-text')).not.toHaveTextContent(
      Messages.storageClassesNoReadPermission
    );
  });
});
