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
import { screen, render } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { DatabasePreview } from './database-preview.tsx';
import { DbWizardType } from '../database-form-schema.ts';
import { DatabaseFormProvider } from '../database-form-context.tsx';
import { Section } from 'components/ui-generator/ui-generator.types.ts';

const FormProviderWrapper = ({
  children,
  defaultValues = {},
}: {
  children: React.ReactNode;
  defaultValues?: Partial<DbWizardType>;
}) => {
  const methods = useForm<DbWizardType>({
    defaultValues: defaultValues as DbWizardType,
  });

  return (
    <FormProvider {...methods}>
      <form>{children}</form>
    </FormProvider>
  );
};

describe('DatabasePreview - Sections Order', () => {
  it('should render sections in the order specified by sectionsOrder', () => {
    const sections: { [key: string]: Section } = {
      databaseVersion: {
        label: 'Database Version',
        components: {},
      },
      resources: {
        label: 'Resources',
        components: {},
      },
      advancedConfigurations: {
        label: 'Advanced Configurations',
        components: {},
      },
    };

    // Define the order explicitly
    const sectionsOrder = [
      'databaseVersion',
      'resources',
      'advancedConfigurations',
    ];

    render(
      <FormProviderWrapper>
        <TestWrapper>
          <DatabaseFormProvider
            value={{
              uiSchema: {},
              topologies: ['replica'],
              hasMultipleTopologies: false,
              defaultTopology: 'replica',
              sections,
              sectionsOrder,
              providerObject: undefined,
              hasBackupStep: false,
            }}
          >
            <DatabasePreview stepsWithErrors={[]} activeStepId="base" />
          </DatabaseFormProvider>
        </TestWrapper>
      </FormProviderWrapper>
    );

    // Get all section titles
    const basicInfo = screen.getByText('1. Basic Information');
    const dbVersion = screen.getByText('2. Database Version');
    const resources = screen.getByText('3. Resources');
    const advancedConfig = screen.getByText('4. Advanced Configurations');

    expect(basicInfo).toBeInTheDocument();
    expect(dbVersion).toBeInTheDocument();
    expect(resources).toBeInTheDocument();
    expect(advancedConfig).toBeInTheDocument();

    // Verify the order by checking their positions in the DOM
    const allSections = screen.getAllByText(/^\d+\./);
    expect(allSections[0]).toHaveTextContent('1. Basic Information');
    expect(allSections[1]).toHaveTextContent('2. Database Version');
    expect(allSections[2]).toHaveTextContent('3. Resources');
    expect(allSections[3]).toHaveTextContent('4. Advanced Configurations');
  });

  it('should render sections in a different order when sectionsOrder changes', () => {
    const sections: { [key: string]: Section } = {
      databaseVersion: {
        label: 'Database Version',
        components: {},
      },
      resources: {
        label: 'Resources',
        components: {},
      },
      advancedConfigurations: {
        label: 'Advanced Configurations',
        components: {},
      },
    };

    // Different order - resources first, then advancedConfigurations, then databaseVersion
    const sectionsOrder = [
      'resources',
      'advancedConfigurations',
      'databaseVersion',
    ];

    render(
      <FormProviderWrapper>
        <TestWrapper>
          <DatabaseFormProvider
            value={{
              uiSchema: {},
              topologies: ['replica'],
              hasMultipleTopologies: false,
              defaultTopology: 'replica',
              sections,
              sectionsOrder,
              providerObject: undefined,
              hasBackupStep: false,
            }}
          >
            <DatabasePreview stepsWithErrors={[]} activeStepId="base" />
          </DatabaseFormProvider>
        </TestWrapper>
      </FormProviderWrapper>
    );

    // Verify the order by checking their positions in the DOM
    const allSections = screen.getAllByText(/^\d+\./);
    expect(allSections[0]).toHaveTextContent('1. Basic Information');
    expect(allSections[1]).toHaveTextContent('2. Resources');
    expect(allSections[2]).toHaveTextContent('3. Advanced Configurations');
    expect(allSections[3]).toHaveTextContent('4. Database Version');
  });

  it('should fall back to Object.keys order when sectionsOrder is not provided', () => {
    const sections: { [key: string]: Section } = {
      databaseVersion: {
        label: 'Database Version',
        components: {},
      },
      resources: {
        label: 'Resources',
        components: {},
      },
      advancedConfigurations: {
        label: 'Advanced Configurations',
        components: {},
      },
    };

    render(
      <FormProviderWrapper>
        <TestWrapper>
          <DatabaseFormProvider
            value={{
              uiSchema: {},
              topologies: ['replica'],
              hasMultipleTopologies: false,
              defaultTopology: 'replica',
              sections,
              sectionsOrder: undefined,
              providerObject: undefined,
              hasBackupStep: false,
            }}
          >
            <DatabasePreview stepsWithErrors={[]} activeStepId="base" />
          </DatabaseFormProvider>
        </TestWrapper>
      </FormProviderWrapper>
    );

    // Should have all sections present (order may vary based on Object.keys)
    expect(screen.getByText('1. Basic Information')).toBeInTheDocument();
    expect(screen.getByText(/Database Version/)).toBeInTheDocument();
    expect(screen.getByText(/Resources/)).toBeInTheDocument();
    expect(screen.getByText(/Advanced Configurations/)).toBeInTheDocument();
  });
});
