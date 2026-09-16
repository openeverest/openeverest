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

import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { useEffect } from 'react';
import { SwitchInput } from '@percona/ui-lib';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import {
  Component,
  FieldType,
  FormMode,
  ToggleFieldParams,
  TopologyUISchemas,
} from '../ui-generator.types';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { Button } from '@mui/material';
import { postprocessSchemaData } from '../utils/postprocess/postprocess-schema';
import { UiGeneratorProvider } from '../ui-generator-context';
import { applyModeOverrides } from '../utils/preprocess/apply-mode-overrides';

vi.mock('../utils/schema-builder/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

/** MUI Switch exposes the native input as role="switch" with the field label as name. */
const getTestToggleSwitch = () =>
  screen.getByRole('switch', { name: /Test Toggle Field/i });

const createTestSchema = (
  fieldParams: Partial<ToggleFieldParams> = {},
  componentOverrides: {
    validation?: Extract<Component, { uiType: FieldType.Toggle }>['validation'];
    modes?: Extract<Component, { uiType: FieldType.Toggle }>['modes'];
  } = {}
): TopologyUISchemas => {
  const testToggle: Extract<Component, { uiType: FieldType.Toggle }> = {
    uiType: FieldType.Toggle,
    path: 'spec.testToggle',
    fieldParams: {
      label: 'Test Toggle Field',
      ...fieldParams,
    },
    ...componentOverrides,
  };

  return {
    testTopology: {
      sections: {
        basicInfo: {
          label: 'Basic Information',
          components: { testToggle },
        },
      },
      sectionsOrder: ['basicInfo'],
    },
  };
};

interface FormWrapperProps {
  children: React.ReactNode;
  schema: TopologyUISchemas;
  onSubmit?: (data: Record<string, unknown>) => void;
  formMode?: FormMode;
}

const FormWrapper = ({
  children,
  schema,
  onSubmit = vi.fn(),
  formMode,
}: FormWrapperProps) => {
  const { schema: zodSchema } = buildZodSchema(schema, 'testTopology');
  const defaultValues = getDefaultValues(schema, 'testTopology');

  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues,
    reValidateMode: 'onChange',
  });

  return (
    <UiGeneratorProvider formMode={formMode}>
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(onSubmit)}>
          {children}
          <Button type="submit" data-testid="submit-button">
            Submit
          </Button>
        </form>
      </FormProvider>
    </UiGeneratorProvider>
  );
};

/**
 * Renders a bare SwitchInput and pushes a field-level error (as CEL validation
 * would) so we can assert the toggle surfaces validation errors to the user.
 */
const ToggleErrorHarness = ({ errorMessage }: { errorMessage: string }) => {
  const methods = useForm({
    defaultValues: { spec: { testToggle: false } },
  });

  useEffect(() => {
    methods.setError('spec.testToggle', { type: 'cel', message: errorMessage });
  }, [errorMessage, methods]);

  return (
    <FormProvider {...methods}>
      <SwitchInput name="spec.testToggle" label="Test Toggle Field" />
    </FormProvider>
  );
};

describe('UIGenerator - Toggle Field Basic Rendering', () => {
  it('should render a switch with the correct label', () => {
    const schema = createTestSchema();

    render(
      <TestWrapper>
        <FormWrapper schema={schema}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByText('Test Toggle Field')).toBeInTheDocument();
    expect(
      screen.getByTestId('switch-input-spec.test-toggle')
    ).toBeInTheDocument();
  });

  it('should default to unchecked when no defaultValue is set', () => {
    const schema = createTestSchema();

    render(
      <TestWrapper>
        <FormWrapper schema={schema}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(getTestToggleSwitch()).not.toBeChecked();
  });

  it('should render checked when defaultValue is true', () => {
    const schema = createTestSchema({ defaultValue: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(getTestToggleSwitch()).toBeChecked();
  });

  it('should render helperText as a caption under the label', () => {
    const schema = createTestSchema({
      helperText: 'Optional helper caption',
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByText('Optional helper caption')).toBeInTheDocument();
  });

  it('should render disabled state', () => {
    const schema = createTestSchema({ disabled: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(getTestToggleSwitch()).toBeDisabled();
  });

  it('should autofocus the switch when autoFocus is set', () => {
    const schema = createTestSchema({ autoFocus: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(getTestToggleSwitch()).toHaveFocus();
  });

  it('should toggle its value on click and submit the new boolean', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema();

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const toggle = getTestToggleSwitch();
    expect(toggle).not.toBeChecked();

    fireEvent.click(toggle);
    expect(toggle).toBeChecked();

    fireEvent.click(screen.getByTestId('submit-button'));

    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ spec: { testToggle: true } }),
        expect.anything()
      );
    });
  });

  it('should render disabled via fieldParams mode override in edit mode', () => {
    const schema = createTestSchema({
      modes: { [FormMode.Edit]: { disabled: true } },
    });
    const sections = applyModeOverrides(
      schema.testTopology!.sections,
      FormMode.Edit
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} formMode={FormMode.Edit}>
          <UIGenerator sections={sections} sectionKey="basicInfo" />
        </FormWrapper>
      </TestWrapper>
    );

    expect(getTestToggleSwitch()).toBeDisabled();
  });

  it('should not render when component mode overrides uiType to hidden', () => {
    const schema = createTestSchema(
      {},
      { modes: { [FormMode.Edit]: { uiType: 'hidden' } } }
    );
    const sections = applyModeOverrides(
      schema.testTopology!.sections,
      FormMode.Edit
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} formMode={FormMode.Edit}>
          <UIGenerator sections={sections} sectionKey="basicInfo" />
        </FormWrapper>
      </TestWrapper>
    );

    expect(
      screen.queryByTestId('switch-input-spec.test-toggle')
    ).not.toBeInTheDocument();
  });
});

describe('UIGenerator - Toggle Field submit and postprocess', () => {
  it('should submit boolean values in nested form data', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({ defaultValue: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    fireEvent.click(screen.getByTestId('submit-button'));

    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          spec: { testToggle: true },
        }),
        expect.anything()
      );
    });
  });

  it('should preserve false values during postprocess', () => {
    const input = {
      spec: { testToggle: false, notes: '' },
    };

    const result = postprocessSchemaData(input);

    expect(result).toEqual({
      spec: { testToggle: false },
    });
  });
});

describe('UIGenerator - Toggle Field validation', () => {
  it('should ignore validation.required for toggle fields', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        validation: {
          // `required` is excluded from ToggleValidation at the type level;
          // simulate an untyped YAML schema that still sets it to prove the
          // runtime ignores it for toggles.
          // @ts-expect-error required is not part of ToggleValidation
          required: true,
        },
      }
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    fireEvent.click(screen.getByTestId('submit-button'));

    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalled();
    });
  });

  it('should surface field validation (CEL) errors below the switch', async () => {
    render(
      <TestWrapper>
        <ToggleErrorHarness errorMessage="Monitoring must be enabled for this tier" />
      </TestWrapper>
    );

    expect(
      await screen.findByText('Monitoring must be enabled for this tier')
    ).toBeInTheDocument();
  });
});
