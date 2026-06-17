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
import { render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { TestWrapper } from 'utils/test';
import { WizardMode } from 'shared-types/wizard.types';
import { DbWizardFormFields } from 'consts';
import { DatabaseFormProvider } from 'pages/database-form/database-form-context';
import { BaseInfoStep } from './base-step';
import { PreviewSectionOne } from 'pages/database-form/database-preview/sections/base-step';
import {
  getDBWizardSchema,
  DbWizardTypeBase,
} from 'pages/database-form/database-form-schema';
import { z } from 'zod';

vi.mock('hooks/api/namespaces/useNamespaces', () => ({
  useNamespaces: vi.fn(() => ({
    data: ['test-namespace', 'another-namespace'],
    isLoading: false,
  })),
}));

vi.mock('hooks/rbac', () => ({
  useNamespacePermissionsForResource: vi.fn(() => ({
    isLoading: false,
    canCreate: ['test-namespace', 'another-namespace'],
  })),
}));

vi.mock('../../../hooks/use-database-page-mode', () => ({
  useDatabasePageMode: vi.fn(() => WizardMode.New),
}));

const makeDefaultValues = (topologyType = 'replica') => ({
  [DbWizardFormFields.provider]: 'psmdb',
  [DbWizardFormFields.dbName]: 'my-test-db',
  [DbWizardFormFields.k8sNamespace]: 'test-namespace',
  topology: { type: topologyType },
});

const makeContextValue = (topologies: string[] = ['replica']) => ({
  uiSchema: {},
  topologies,
  hasMultipleTopologies: topologies.length > 1,
  defaultTopology: topologies[0] ?? '',
  sections: {},
  sectionsOrder: [],
  providerObject: undefined,
  hasBackupStep: false,
});

interface WrapperProps {
  children: React.ReactNode;
  defaultValues?: ReturnType<typeof makeDefaultValues>;
  topologies?: string[];
  schema?: z.ZodTypeAny;
}

const Wrapper = ({
  children,
  defaultValues = makeDefaultValues(),
  topologies = ['replica'],
  schema,
}: WrapperProps) => {
  const methods = useForm({
    mode: 'onChange',
    defaultValues,
    ...(schema ? { resolver: zodResolver(schema) } : {}),
  });

  return (
    <TestWrapper>
      <DatabaseFormProvider value={makeContextValue(topologies)}>
        <FormProvider {...methods}>
          <form>{children}</form>
        </FormProvider>
      </DatabaseFormProvider>
    </TestWrapper>
  );
};

describe('BaseInfoStep', () => {
  it('renders all required base fields on form open', async () => {
    render(
      <Wrapper>
        <BaseInfoStep loadingDefaultsForEdition={false} />
      </Wrapper>
    );

    // Namespace autocomplete
    expect(screen.getByTestId('text-input-k8s-namespace')).toBeInTheDocument();

    // Database name input
    expect(screen.getByTestId('text-input-db-name')).toBeInTheDocument();
  });

  it('does NOT show topology select when only one topology is available', () => {
    render(
      <Wrapper topologies={['replica']}>
        <BaseInfoStep loadingDefaultsForEdition={false} />
      </Wrapper>
    );

    expect(
      screen.queryByTestId('select-input-topology.type')
    ).not.toBeInTheDocument();
  });

  it('shows topology select when multiple topologies are provided', async () => {
    render(
      <Wrapper topologies={['replica', 'sharded']}>
        <BaseInfoStep loadingDefaultsForEdition={false} />
      </Wrapper>
    );

    await waitFor(() =>
      expect(
        screen.getByTestId('select-input-topology.type')
      ).toBeInTheDocument()
    );
  });

  it('topology select pre-fills with the default topology value', async () => {
    render(
      <Wrapper
        topologies={['replica', 'sharded']}
        defaultValues={makeDefaultValues('replica')}
      >
        <BaseInfoStep loadingDefaultsForEdition={false} />
      </Wrapper>
    );

    await waitFor(() =>
      expect(screen.getByTestId('select-input-topology.type')).toHaveValue(
        'replica'
      )
    );
  });
});

describe('basicInfoSchema (topology as nested object)', () => {
  const schema = getDBWizardSchema([], false);

  it('passes when topology is provided as a nested object { type: string }', () => {
    const result = schema.safeParse(makeDefaultValues('replica'));
    expect(result.success).toBe(true);
  });

  it('fails when topology.type is missing', () => {
    const data = {
      ...makeDefaultValues(),
      topology: {},
    };
    const result = schema.safeParse(data);
    expect(result.success).toBe(false);
  });

  it('fails when topology is a plain string (old behaviour)', () => {
    const data = {
      ...makeDefaultValues(),
      topology: 'replica',
    };
    const result = schema.safeParse(data);
    expect(result.success).toBe(false);
  });
});

describe('PreviewSectionOne (base-step preview)', () => {
  const renderPreview = (overrides: Record<string, unknown> = {}) =>
    render(
      <TestWrapper>
        <PreviewSectionOne
          {...(makeDefaultValues() as unknown as DbWizardTypeBase)}
          {...(overrides as unknown as DbWizardTypeBase)}
        />
      </TestWrapper>
    );

  it('displays namespace when provided', () => {
    renderPreview({ k8sNamespace: 'test-namespace' });
    expect(screen.getByText('Namespace: test-namespace')).toBeInTheDocument();
  });

  it('displays provider when provided', () => {
    renderPreview({ provider: 'psmdb' });
    expect(screen.getByText('Provider: psmdb')).toBeInTheDocument();
  });

  it('displays database name when provided', () => {
    renderPreview({ dbName: 'my-test-db' });
    expect(screen.getByText('Name: my-test-db')).toBeInTheDocument();
  });

  it('displays topology when provided', () => {
    renderPreview({ topology: { type: 'replica' } });
    expect(screen.getByText('Topology: replica')).toBeInTheDocument();
  });

  it('does not display topology when it is not provided', () => {
    renderPreview({ topology: undefined });
    expect(screen.queryByText(/Topology:/)).not.toBeInTheDocument();
  });

  it('renders all fields together correctly (no errors in preview)', () => {
    renderPreview({
      k8sNamespace: 'test-namespace',
      provider: 'psmdb',
      dbName: 'my-test-db',
      topology: { type: 'replica' },
    });
    expect(screen.getByText('Namespace: test-namespace')).toBeInTheDocument();
    expect(screen.getByText('Provider: psmdb')).toBeInTheDocument();
    expect(screen.getByText('Name: my-test-db')).toBeInTheDocument();
    expect(screen.getByText('Topology: replica')).toBeInTheDocument();
  });
});
