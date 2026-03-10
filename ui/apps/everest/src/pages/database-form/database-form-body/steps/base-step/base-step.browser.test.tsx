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

import React from 'react';
import { render } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { TestWrapper } from 'utils/test';
import { WizardMode } from 'shared-types/wizard.types';
import { DbWizardFormFields } from 'consts';
import { DatabaseFormProvider } from 'pages/database-form/database-form-context';
import { BaseInfoStep } from './base-step';
import { getDBWizardSchema } from 'pages/database-form/database-form-schema';
import { page, userEvent } from 'vitest/browser';

vi.mock('hooks/api/namespaces/useNamespaces', () => ({
  useNamespaces: vi.fn(() => ({
    data: ['test-namespace', 'another-namespace'],
    isLoading: false,
  })),
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: vi.fn(() => ({
    canRead: true,
    canUpdate: true,
    canCreate: true,
    canDelete: true,
  })),
  useNamespacePermissionsForResource: vi.fn(() => ({
    isLoading: false,
    canCreate: ['test-namespace', 'another-namespace'],
  })),
  useRBACPermissionRoute: vi.fn(() => true),
}));

vi.mock('../../../hooks/use-database-page-mode', () => ({
  useDatabasePageMode: vi.fn(() => WizardMode.New),
}));

const schema = getDBWizardSchema([], false);

const defaultValues = {
  [DbWizardFormFields.provider]: 'psmdb',
  [DbWizardFormFields.dbName]: 'my-test-db',
  [DbWizardFormFields.k8sNamespace]: '',
  topology: { type: 'replica' },
};

const contextValue = {
  uiSchema: {},
  topologies: ['replica', 'sharded'],
  hasMultipleTopologies: true,
  defaultTopology: 'replica',
  sections: {},
  sectionsOrder: [],
  providerObject: undefined,
};

const Wrapper = ({ children }: { children: React.ReactNode }) => {
  const methods = useForm({
    mode: 'onChange',
    defaultValues,
    resolver: zodResolver(schema),
  });

  return (
    <TestWrapper>
      <DatabaseFormProvider value={contextValue}>
        <FormProvider {...methods}>
          <form>{children}</form>
        </FormProvider>
      </DatabaseFormProvider>
    </TestWrapper>
  );
};

describe('BaseInfoStep (browser mode)', () => {
  it('renders form fields and allows editing db name in real browser', async () => {
    render(
      <Wrapper>
        <BaseInfoStep
          loadingDefaultsForEdition={false}
          alreadyVisited={false}
        />
      </Wrapper>
    );

    await expect
      .element(page.getByTestId('text-input-k8s-namespace'))
      .toBeInTheDocument();

    const dbNameInput = page.getByTestId('text-input-db-name');
    await userEvent.fill(dbNameInput, 'db-browser-mode');

    await expect.element(dbNameInput).toHaveValue('db-browser-mode');
  });
});
