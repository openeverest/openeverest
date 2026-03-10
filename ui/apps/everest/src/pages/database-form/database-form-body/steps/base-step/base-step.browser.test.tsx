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
        <BaseInfoStep loadingDefaultsForEdition={false} alreadyVisited={false} />
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
