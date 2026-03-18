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
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { Component, FieldType, TopologyUISchemas } from '../ui-generator.types';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { Button } from '@mui/material';

vi.mock('../utils/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

vi.mock('../utils/schema-builder/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

const testOptions = [
  { label: 'Option One', value: 'option1' },
  { label: 'Option Two', value: 'option2' },
  { label: 'Option Three', value: 'option3' },
];

const createTestSchema = (
  fieldParams: Record<string, unknown> = {},
  validation?: Record<string, unknown>
): TopologyUISchemas => {
  const testSelect: Extract<Component, { uiType: FieldType.Select }> = {
    uiType: FieldType.Select,
    path: 'spec.testSelect',
    fieldParams: {
      label: 'Test Select Field',
      options: testOptions,
      ...fieldParams,
    },
  };

  if (validation) {
    testSelect.validation = validation;
  }

  return {
    testTopology: {
      sections: {
        basicInfo: {
          label: 'Basic Information',
          components: {
            testSelect,
          },
        },
      },
      sectionsOrder: ['basicInfo'],
    },
  };
};

interface FormWrapperProps {
  children: React.ReactNode;
  schema: TopologyUISchemas;
  onSubmit: (data: Record<string, unknown>) => void;
}

const FormWrapper = ({ children, schema, onSubmit }: FormWrapperProps) => {
  const { schema: zodSchema } = buildZodSchema(schema, 'testTopology');
  const defaultValues = getDefaultValues(schema, 'testTopology');

  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues,
    reValidateMode: 'onChange',
  });

  return (
    <FormProvider {...methods}>
      <form onSubmit={methods.handleSubmit(onSubmit)}>
        {children}
        <Button
          type="submit"
          disabled={!methods.formState.isValid}
          data-testid="submit-button"
        >
          Submit
        </Button>
      </form>
    </FormProvider>
  );
};

describe('UIGenerator - Select Field Basic Rendering', () => {
  it('should render select field with options', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({});

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

    const selectButton = screen.getByTestId('select-spec.test-select-button');
    expect(selectButton).toBeInTheDocument();
  });

  it('should display all options when select is opened', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({});

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

    const selectCombobox = screen.getByRole('combobox', {
      name: /test select field/i,
    });
    fireEvent.mouseDown(selectCombobox);

    await waitFor(() => {
      const listbox = screen.getByRole('listbox');
      expect(listbox).toBeInTheDocument();
    });

    // Check options are rendered (they might be in portal)
    expect(screen.getByText('Option One')).toBeInTheDocument();
    expect(screen.getByText('Option Two')).toBeInTheDocument();
    expect(screen.getByText('Option Three')).toBeInTheDocument();
  });
});

describe('UIGenerator - Select Field Required Validation', () => {
  it('should enable submit button when select field is empty by default (optional)', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({});

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

    // Wait for validation to complete
    await waitFor(() => {
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeEnabled();
    });
  });

  it('should disable submit button when required select field is empty', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({ required: true });

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

    const submitButton = screen.getByTestId('submit-button');
    expect(submitButton).toBeDisabled();
  });

  it('should enable submit button when required select field has a value', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({ required: true });

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

    const selectCombobox = screen.getByRole('combobox', {
      name: /test select field/i,
    });
    fireEvent.mouseDown(selectCombobox);

    // Wait for options to appear
    await waitFor(() => {
      expect(screen.getByRole('listbox')).toBeInTheDocument();
    });

    const option1 = screen.getByText('Option One');
    fireEvent.click(option1);

    await waitFor(() => {
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeEnabled();
    });
  });
});

describe('UIGenerator - Select Field Default Value', () => {
  it('should use empty string as default value when not specified', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({});

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

    const submitButton = screen.getByTestId('submit-button');

    // Wait for form to be valid
    await waitFor(() => {
      expect(submitButton).toBeEnabled();
    });

    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalled();
      const callData = mockSubmit.mock.calls[0][0];
      expect(callData).toMatchObject({
        spec: {
          testSelect: '',
        },
      });
    });
  });

  it('should use specified default value', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({ defaultValue: 'option2' });

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

    // Verify the default value is displayed
    await waitFor(() => {
      const selectInput = screen.getByTestId('select-input-spec.test-select');
      expect(selectInput).toHaveValue('option2');
    });
  });
});

describe('UIGenerator - Select Field Enum Validation', () => {
  it('should accept value from the options list', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({ required: true });

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

    const selectCombobox = screen.getByRole('combobox', {
      name: /test select field/i,
    });
    fireEvent.mouseDown(selectCombobox);

    // Wait for options to appear in the portal
    await waitFor(() => {
      expect(screen.getByRole('listbox')).toBeInTheDocument();
    });

    const option2 = screen.getByText('Option Two');
    fireEvent.click(option2);

    await waitFor(() => {
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeEnabled();
    });

    const submitButton = screen.getByTestId('submit-button');
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalled();
      const callData = mockSubmit.mock.calls[0][0];
      expect(callData).toMatchObject({
        spec: {
          testSelect: 'option2',
        },
      });
    });
  });

  it('should display error message for invalid enum value (programmatically set)', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({ defaultValue: 'invalid-option' });

    const TestComponent = () => {
      const { schema: zodSchema } = buildZodSchema(schema, 'testTopology');
      const defaultValues = getDefaultValues(schema, 'testTopology');

      const methods = useForm({
        resolver: zodResolver(zodSchema),
        mode: 'onChange',
        defaultValues,
        reValidateMode: 'onChange',
      });

      return (
        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit(mockSubmit)}>
            <UIGenerator
              sections={schema.testTopology!.sections}
              sectionKey="basicInfo"
            />
            <Button
              type="submit"
              disabled={!methods.formState.isValid}
              data-testid="submit-button"
            >
              Submit
            </Button>
          </form>
        </FormProvider>
      );
    };

    render(
      <TestWrapper>
        <TestComponent />
      </TestWrapper>
    );

    // The submit button should be disabled because the default value is invalid
    await waitFor(() => {
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeDisabled();
    });
  });
});

describe('UIGenerator - Select Field Disabled State', () => {
  it('should render select field as disabled when disabled prop is true', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({ disabled: true });

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

    const selectInput = screen.getByTestId('select-input-spec.test-select');

    expect(selectInput).toBeDisabled();
  });
});

describe('UIGenerator - Select Field Auto-injected Empty Option', () => {
  it('should allow empty value for optional field with displayEmpty', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      { displayEmpty: true },
      { required: false }
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

    // Should start with empty value
    const selectInput = screen.getByTestId('select-input-spec.test-select');
    expect(selectInput).toHaveValue('');

    // Wait for form validation to complete - form should be valid since field is optional
    await waitFor(() => {
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeEnabled();
    });
  });

  it('should NOT allow empty value for required field even with displayEmpty', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({ displayEmpty: true }, { required: true });

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

    // Should start with empty value
    const selectInput = screen.getByTestId('select-input-spec.test-select');
    expect(selectInput).toHaveValue('');

    // Form should be INVALID since field is required
    const submitButton = screen.getByTestId('submit-button');
    expect(submitButton).toBeDisabled();
  });

  it('should maintain empty value when optional field with displayEmpty has no default', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      { displayEmpty: true },
      { required: false }
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

    const selectInput = screen.getByTestId('select-input-spec.test-select');
    expect(selectInput).toHaveValue('');

    // Wait for form to become valid
    const submitButton = screen.getByTestId('submit-button');
    await waitFor(() => {
      expect(submitButton).toBeEnabled();
    });

    // Submit with empty value
    fireEvent.click(submitButton);

    // The transform converts "" to undefined in the submitted data
    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalled();
    });

    // Verify the submitted data structure
    expect(mockSubmit.mock.calls[0][0]).toMatchObject({
      spec: expect.any(Object),
    });
  });

  it('should work without displayEmpty for optional fields', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      { displayEmpty: false },
      { required: false }
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

    // Should still allow empty value for optional field
    const selectInput = screen.getByTestId('select-input-spec.test-select');
    expect(selectInput).toHaveValue('');

    // Wait for form validation to complete
    await waitFor(() => {
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeEnabled();
    });
  });

  it('should respect default value in optional field with displayEmpty', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      { displayEmpty: true, defaultValue: 'option2' },
      { required: false }
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

    // Should have default value
    const selectInput = screen.getByTestId('select-input-spec.test-select');
    expect(selectInput).toHaveValue('option2');

    // Wait for form validation to complete
    await waitFor(() => {
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeEnabled();
    });
  });

  it('should fail validation when required field has no defaultValue and displayEmpty=false', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      { displayEmpty: false }, // No displayEmpty, no defaultValue
      { required: true }
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

    // Field should start with empty value ''
    const selectInput = screen.getByTestId('select-input-spec.test-select');
    expect(selectInput).toHaveValue('');

    // Form should be INVALID since field is required and empty
    const submitButton = screen.getByTestId('submit-button');
    expect(submitButton).toBeDisabled();

    // Try to submit - should not call the submit handler
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockSubmit).not.toHaveBeenCalled();
    });
  });
});
