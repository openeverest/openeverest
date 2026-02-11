import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { TopologyUISchemas } from '../ui-generator.types';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { Button } from '@mui/material';

vi.mock('../utils/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

const createTestSchema = (
  fieldParams: any = {},
  validation?: any
): Partial<TopologyUISchemas> => {
  const testNumber: any = {
    uiType: 'number',
    path: 'spec.testNumber',
    fieldParams: {
      label: 'Test Number Field',
      ...fieldParams,
    },
  };

  if (validation) {
    testNumber.validation = validation;
  }

  return {
    testTopology: {
      sections: {
        basicInfo: {
          label: 'Basic Information',
          components: {
            testNumber,
          },
        },
      },
      sectionsOrder: ['basicInfo'],
    },
  };
};

interface FormWrapperProps {
  children: React.ReactNode;
  schema: Partial<TopologyUISchemas>;
  onSubmit: (data: any) => void;
}

const FormWrapper = ({ children, schema, onSubmit }: FormWrapperProps) => {
  const { schema: zodSchema } = buildZodSchema(schema, 'testTopology');

  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues: {},
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
          Submit// Field is required
        </Button>
      </form>
    </FormProvider>
  );
};

describe('UIGenerator - Number Field Required Validation', () => {
  it('should disable submit button when required number field is empty', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      required: true,
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const submitButton = screen.getByTestId('submit-button');
    const numberInput = screen.getByLabelText('Test Number Field');

    await waitFor(() => {
      expect(submitButton).toBeDisabled();
    });
    expect(numberInput).toBeInTheDocument();
  });

  it('should enable submit button when optional number field is empty', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      required: false,
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const submitButton = screen.getByTestId('submit-button');
    const numberInput = screen.getByLabelText('Test Number Field');

    expect(numberInput).toBeInTheDocument();

    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });
  });

  it('should show "Field is required" error for required empty field', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      required: true,
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const submitButton = screen.getByTestId('submit-button');

    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockSubmit).not.toHaveBeenCalled();
      expect(submitButton).toBeDisabled();
    });
  });

  it('should not show error for optional empty field', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      required: false,
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const numberInput = screen.getByLabelText('Test Number Field');

    numberInput.focus();
    numberInput.blur();

    await waitFor(() => {
      expect(screen.queryByText('Field is required')).not.toBeInTheDocument();
    });
  });
});

describe('UIGenerator - Number Field Label and Helper Text', () => {
  it('should render label and helper text when provided in schema', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      label: 'Number Field',
      helperText: 'This is a helpful description',
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByLabelText('Number Field')).toBeInTheDocument();
    expect(
      screen.getByText('This is a helpful description')
    ).toBeInTheDocument();
  });
});

describe('UIGenerator - Number Field Disabled State', () => {
  it('should disable the field when disabled param is set in schema', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      disabled: true,
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const numberInput = screen.getByLabelText('Test Number Field');
    expect(numberInput).toBeDisabled();
  });
});

describe('UIGenerator - Number Field Min/Max Validation', () => {
  it('should show error when value is less than min', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        min: 5,
        max: 10,
      }
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const numberInput = screen.getByLabelText('Test Number Field');

    fireEvent.change(numberInput, { target: { value: '3' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.getByText('Number must be greater than or equal to 5')
      ).toBeInTheDocument();
    });
  });

  it('should show error when value is greater than max', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        min: 5,
        max: 10,
      }
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const numberInput = screen.getByLabelText('Test Number Field');

    fireEvent.change(numberInput, { target: { value: '15' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.getByText('Number must be less than or equal to 10')
      ).toBeInTheDocument();
    });
  });

  it('should accept valid value within min/max range', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        min: 5,
        max: 10,
      }
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const numberInput = screen.getByLabelText('Test Number Field');
    const submitButton = screen.getByTestId('submit-button');

    fireEvent.change(numberInput, { target: { value: '7' } });
    numberInput.blur();

    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Number Field Input Attributes from Validation', () => {
  it('should apply min/max from validation to input attributes', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        min: 5,
        max: 10,
      }
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const numberInput = screen.getByLabelText(
      'Test Number Field'
    ) as HTMLInputElement;

    expect(numberInput.min).toBe('5');
    expect(numberInput.max).toBe('10');
  });
});

describe('UIGenerator - Number Field Default Value', () => {
  it('should populate default value on initial render', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      defaultValue: 42,
    });

    const FormWithDefaults = ({
      children,
      schema,
      onSubmit,
    }: FormWrapperProps) => {
      const { schema: zodSchema } = buildZodSchema(schema, 'testTopology');
      const defaultValues = getDefaultValues(schema, 'testTopology');

      const methods = useForm({
        resolver: zodResolver(zodSchema),
        mode: 'onChange',
        defaultValues,
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

    render(
      <TestWrapper>
        <FormWithDefaults schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWithDefaults>
      </TestWrapper>
    );

    const numberInput = screen.getByLabelText(
      'Test Number Field'
    ) as HTMLInputElement;

    expect(numberInput.value).toBe('42');
  });
});

describe('UIGenerator - Number Field Step Param', () => {
  it('should set step attribute when step param is provided', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      step: 2,
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const numberInput = screen.getByLabelText(
      'Test Number Field'
    ) as HTMLInputElement;

    expect(numberInput.step).toBe('2');
  });
});

describe('UIGenerator - Number Field Placeholder', () => {
  it('should display placeholder when field is empty and placeholder is provided', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({
      placeholder: 'Enter a number',
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            activeStep={0}
            sections={schema.testTopology.sections}
            stepLabels={['basicInfo']}
          />
        </FormWrapper>
      </TestWrapper>
    );

    const numberInput = screen.getByPlaceholderText('Enter a number');

    expect(numberInput).toBeInTheDocument();
  });
});

describe('UIGenerator - Number Field Advanced Validation', () => {
  describe('Integer validation', () => {
    it('should reject decimal values when int validation is set', async () => {
      const mockSubmit = vi.fn();
      const schema = createTestSchema(
        {},
        {
          int: true,
        }
      );

      render(
        <TestWrapper>
          <FormWrapper schema={schema} onSubmit={mockSubmit}>
            <UIGenerator
              activeStep={0}
              sections={schema.testTopology.sections}
              stepLabels={['basicInfo']}
            />
          </FormWrapper>
        </TestWrapper>
      );

      const numberInput = screen.getByLabelText('Test Number Field');

      fireEvent.change(numberInput, { target: { value: '3.5' } });
      numberInput.blur();

      await waitFor(() => {
        expect(
          screen.getByText('Expected integer, received float')
        ).toBeInTheDocument();
      });
    });

    it('should accept integer values when int validation is set', async () => {
      const mockSubmit = vi.fn();
      const schema = createTestSchema(
        {},
        {
          int: true,
        }
      );

      render(
        <TestWrapper>
          <FormWrapper schema={schema} onSubmit={mockSubmit}>
            <UIGenerator
              activeStep={0}
              sections={schema.testTopology.sections}
              stepLabels={['basicInfo']}
            />
          </FormWrapper>
        </TestWrapper>
      );

      const numberInput = screen.getByLabelText('Test Number Field');
      const submitButton = screen.getByTestId('submit-button');

      fireEvent.change(numberInput, { target: { value: '5' } });
      numberInput.blur();

      await waitFor(() => {
        expect(submitButton).not.toBeDisabled();
      });
    });
  });

  describe('GT/LT validation (exclusive bounds)', () => {
    it('should enforce gt (greater than) validation', async () => {
      const mockSubmit = vi.fn();
      const schema = createTestSchema(
        {},
        {
          gt: 5,
        }
      );

      render(
        <TestWrapper>
          <FormWrapper schema={schema} onSubmit={mockSubmit}>
            <UIGenerator
              activeStep={0}
              sections={schema.testTopology.sections}
              stepLabels={['basicInfo']}
            />
          </FormWrapper>
        </TestWrapper>
      );

      const numberInput = screen.getByLabelText('Test Number Field');

      fireEvent.change(numberInput, { target: { value: '5' } });
      numberInput.blur();

      await waitFor(() => {
        expect(
          screen.getByText('Number must be greater than 5')
        ).toBeInTheDocument();
      });
    });

    it('should enforce lt (less than) validation', async () => {
      const mockSubmit = vi.fn();
      const schema = createTestSchema(
        {},
        {
          lt: 10,
        }
      );

      render(
        <TestWrapper>
          <FormWrapper schema={schema} onSubmit={mockSubmit}>
            <UIGenerator
              activeStep={0}
              sections={schema.testTopology.sections}
              stepLabels={['basicInfo']}
            />
          </FormWrapper>
        </TestWrapper>
      );

      const numberInput = screen.getByLabelText('Test Number Field');

      fireEvent.change(numberInput, { target: { value: '10' } });
      numberInput.blur();

      await waitFor(() => {
        expect(
          screen.getByText('Number must be less than 10')
        ).toBeInTheDocument();
      });
    });
  });

  describe('MultipleOf validation', () => {
    it('should reject values not multiple of specified number', async () => {
      const mockSubmit = vi.fn();
      const schema = createTestSchema(
        {},
        {
          multipleOf: 5,
        }
      );

      render(
        <TestWrapper>
          <FormWrapper schema={schema} onSubmit={mockSubmit}>
            <UIGenerator
              activeStep={0}
              sections={schema.testTopology.sections}
              stepLabels={['basicInfo']}
            />
          </FormWrapper>
        </TestWrapper>
      );

      const numberInput = screen.getByLabelText('Test Number Field');

      fireEvent.change(numberInput, { target: { value: '7' } });
      numberInput.blur();

      await waitFor(() => {
        expect(
          screen.getByText('Number must be a multiple of 5')
        ).toBeInTheDocument();
      });
    });

    it('should accept values that are multiple of specified number', async () => {
      const mockSubmit = vi.fn();
      const schema = createTestSchema(
        {},
        {
          multipleOf: 5,
        }
      );

      render(
        <TestWrapper>
          <FormWrapper schema={schema} onSubmit={mockSubmit}>
            <UIGenerator
              activeStep={0}
              sections={schema.testTopology.sections}
              stepLabels={['basicInfo']}
            />
          </FormWrapper>
        </TestWrapper>
      );

      const numberInput = screen.getByLabelText('Test Number Field');
      const submitButton = screen.getByTestId('submit-button');

      fireEvent.change(numberInput, { target: { value: '15' } });
      numberInput.blur();

      await waitFor(() => {
        expect(submitButton).not.toBeDisabled();
      });
    });
  });
});
