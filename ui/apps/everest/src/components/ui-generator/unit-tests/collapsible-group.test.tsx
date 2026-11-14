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
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from '@mui/material';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { FieldType, GroupType, TopologyUISchemas } from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';

const TOPOLOGY = 'replicaSet';

const buildSchema = (
  groupType: GroupType.Collapsible | GroupType.Accordion,
  options?: { label?: string; defaultExpanded?: boolean }
): TopologyUISchemas =>
  preprocessSchema({
    [TOPOLOGY]: {
      sections: {
        advanced: {
          components: {
            advancedSettings: {
              uiType: 'group',
              groupType,
              ...(options?.label !== undefined
                ? { label: options.label }
                : { label: 'Advanced Settings' }),
              description: 'Optional advanced configuration',
              groupParams: {
                defaultExpanded: options?.defaultExpanded ?? false,
              },
              components: {
                setting1: {
                  uiType: FieldType.Text,
                  path: 'spec.advanced.setting1',
                  fieldParams: { label: 'Setting 1' },
                  validation: { required: true },
                },
              },
              componentsOrder: ['setting1'],
            },
          },
        },
      },
    },
  });

const FormWrapper = ({
  schema,
}: {
  schema: TopologyUISchemas;
}) => {
  const { schema: zodSchema } = buildZodSchema(schema, TOPOLOGY);
  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues: getDefaultValues(schema, TOPOLOGY),
  });

  return (
    <FormProvider {...methods}>
      <form
        onSubmit={methods.handleSubmit(() => undefined)}
      >
        <UIGenerator
          sections={schema[TOPOLOGY].sections}
          sectionKey="advanced"
        />
        <Button type="submit">Submit</Button>
      </form>
    </FormProvider>
  );
};

const renderCollapsible = (
  groupType: GroupType.Collapsible | GroupType.Accordion = GroupType.Collapsible,
  options?: { label?: string; defaultExpanded?: boolean }
) => {
  const schema = buildSchema(groupType, options);
  render(
    <TestWrapper>
      <FormWrapper schema={schema} />
    </TestWrapper>
  );
  return schema;
};

const getSummary = () => {
  const summaries = screen.getAllByRole('button');
  const summary = summaries.find((el) =>
    el.classList.contains('MuiAccordionSummary-root')
  );
  if (!summary) {
    throw new Error('Accordion summary not found');
  }
  return summary;
};

describe('UIGenerator - collapsible group', () => {
  it('starts collapsed when defaultExpanded is false', () => {
    renderCollapsible();

    expect(getSummary()).toHaveAttribute('aria-expanded', 'false');
    expect(
      screen.getByText('Optional advanced configuration')
    ).toBeInTheDocument();
  });

  it('keeps field values when collapsed and expanded again', async () => {
    renderCollapsible(GroupType.Collapsible, { defaultExpanded: true });

    const input = screen.getByLabelText('Setting 1');
    fireEvent.change(input, { target: { value: 'kept-value' } });
    expect(input).toHaveValue('kept-value');

    fireEvent.click(getSummary());
    await waitFor(() =>
      expect(getSummary()).toHaveAttribute('aria-expanded', 'false')
    );

    fireEvent.click(getSummary());
    await waitFor(() =>
      expect(getSummary()).toHaveAttribute('aria-expanded', 'true')
    );
    expect(screen.getByLabelText('Setting 1')).toHaveValue('kept-value');
  });

  it('shows a header error icon when collapsed with an invalid child', async () => {
    renderCollapsible(GroupType.Collapsible, { defaultExpanded: true });

    fireEvent.click(screen.getByRole('button', { name: 'Submit' }));
    await waitFor(() => {
      expect(screen.getByLabelText('Setting 1')).toBeInvalid();
    });

    fireEvent.click(getSummary());
    await waitFor(() =>
      expect(getSummary()).toHaveAttribute('aria-expanded', 'false')
    );

    expect(screen.getByTestId('collapsible-group-error')).toBeInTheDocument();
  });

  it('always renders a summary chevron even without a label', () => {
    renderCollapsible(GroupType.Collapsible, { label: '' });

    expect(getSummary()).toBeInTheDocument();
    expect(
      getSummary().querySelector('.MuiAccordionSummary-expandIconWrapper')
    ).not.toBeNull();
  });

  it('accepts deprecated groupType accordion as an alias', () => {
    renderCollapsible(GroupType.Accordion, { defaultExpanded: false });

    expect(screen.getByText('Advanced Settings')).toBeInTheDocument();
    expect(getSummary()).toHaveAttribute('aria-expanded', 'false');
  });
});
