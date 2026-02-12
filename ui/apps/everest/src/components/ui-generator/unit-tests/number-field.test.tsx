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

vi.mock('../utils/schema-builder/cel-validation', () => ({
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
  const defaultValues = getDefaultValues(
    schema.testTopology?.sections || {},
    'testTopology'
  );

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
          Submit// Field is required
        </Button>
      </form>
    </FormProvider>
  );
};

describe('UIGenerator - Number Field Required Validation', () => {
  it('should enable submit button when number field is empty by default (optional)', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({});

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

  it('should not show error for default empty field (optional by default)', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema({});

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

  it('should show "Field is required" error for explicitly required empty field', async () => {
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

  it('should convert gt (greater than) to min attribute for integers', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        gt: 5,
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

    const numberInput = screen.getByLabelText(
      'Test Number Field'
    ) as HTMLInputElement;

    // gt: 5 with int validation should become min: 6
    expect(numberInput.min).toBe('6');
  });

  it('should convert lt (less than) to max attribute for integers', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        lt: 10,
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

    const numberInput = screen.getByLabelText(
      'Test Number Field'
    ) as HTMLInputElement;

    // lt: 10 with int validation should become max: 9
    expect(numberInput.max).toBe('9');
  });

  it('should convert gt/lt using step value for decimals', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {
        step: 0.5,
      },
      {
        gt: 5,
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

    const numberInput = screen.getByLabelText(
      'Test Number Field'
    ) as HTMLInputElement;

    // gt: 5 with step: 0.5 should become min: 5.5
    expect(numberInput.min).toBe('5.5');
    // lt: 10 with step: 0.5 should become max: 9.5
    expect(numberInput.max).toBe('9.5');
  });

  it('should prioritize explicit min/max over gt/lt', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        min: 3,
        max: 12,
        gt: 5,
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

    const numberInput = screen.getByLabelText(
      'Test Number Field'
    ) as HTMLInputElement;

    // Explicit min/max should take priority
    expect(numberInput.min).toBe('3');
    expect(numberInput.max).toBe('12');
  });

  it('should use small offset for gt/lt without int validation or step', () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        gt: 5,
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

    const numberInput = screen.getByLabelText(
      'Test Number Field'
    ) as HTMLInputElement;

    // Should use small offset (0.000001) for arbitrary decimals
    expect(parseFloat(numberInput.min)).toBeCloseTo(5.000001, 6);
    expect(parseFloat(numberInput.max)).toBeCloseTo(9.999999, 6);
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

describe('UIGenerator - Optional Field with Validation', () => {
  it('should not show validation error when optional field with min/max is empty', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {
        required: false,
      },
      {
        min: 1,
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

    const submitButton = screen.getByTestId('submit-button');
    const numberInput = screen.getByLabelText('Test Number Field');

    // Field should start empty
    expect(numberInput).toHaveValue(null);

    // Form should be valid even with empty optional field
    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    // Focus and blur should not trigger validation errors
    numberInput.focus();
    numberInput.blur();

    await waitFor(() => {
      expect(screen.queryByText(/Number must be/)).not.toBeInTheDocument();
      expect(screen.queryByText('Field is required')).not.toBeInTheDocument();
    });

    // Submit should work with empty optional field
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalled();
    });
  });

  it('should validate min/max when optional field has a value', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {
        required: false,
      },
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

    // Enter value below min
    fireEvent.change(numberInput, { target: { value: '3' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.getByText('Number must be greater than or equal to 5')
      ).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });

    // Enter value above max
    fireEvent.change(numberInput, { target: { value: '15' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.getByText('Number must be less than or equal to 10')
      ).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });

    // Enter valid value
    fireEvent.change(numberInput, { target: { value: '7' } });
    numberInput.blur();

    await waitFor(() => {
      expect(screen.queryByText(/Number must be/)).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled();
    });
  });

  it('should allow clearing value in optional field with validation', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {
        required: false,
      },
      {
        min: 1,
        max: 100,
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

    // Enter a valid value
    fireEvent.change(numberInput, { target: { value: '50' } });
    numberInput.blur();

    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    // Clear the value
    fireEvent.change(numberInput, { target: { value: '' } });
    numberInput.blur();

    // Should still be valid (no validation errors)
    await waitFor(() => {
      expect(screen.queryByText(/Number must be/)).not.toBeInTheDocument();
      expect(screen.queryByText('Field is required')).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled();
    });
  });

  it('should work correctly with gt/lt validation on optional fields', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {
        required: false,
      },
      {
        gt: 0,
        lt: 100,
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

    // Empty field should be valid
    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    // Value at boundary (should fail - must be greater than)
    fireEvent.change(numberInput, { target: { value: '0' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.getByText('Number must be greater than 0')
      ).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });

    // Value in valid range
    fireEvent.change(numberInput, { target: { value: '50' } });
    numberInput.blur();

    await waitFor(() => {
      expect(screen.queryByText(/Number must be/)).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled();
    });

    // Clear to empty again
    fireEvent.change(numberInput, { target: { value: '' } });
    numberInput.blur();

    await waitFor(() => {
      expect(screen.queryByText(/Number must be/)).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled();
    });
  });

  it('should enforce int validation on optional field only when value is provided', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {
        required: false,
      },
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

    // Empty should be valid
    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    // Decimal should fail
    fireEvent.change(numberInput, { target: { value: '3.5' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.getByText('Expected integer, received float')
      ).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });

    // Integer should pass
    fireEvent.change(numberInput, { target: { value: '5' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.queryByText('Expected integer, received float')
      ).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled();
    });

    // Clear to empty should be valid
    fireEvent.change(numberInput, { target: { value: '' } });
    numberInput.blur();

    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Regex Validation for All Field Types', () => {
  it('should validate number field with regex pattern', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {},
      {
        regex: {
          pattern: '^[1-9][0-9]*$', // Positive integers only
          message: 'Must be a positive integer',
        },
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

    // Invalid: starts with 0
    fireEvent.change(numberInput, { target: { value: '0123' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.getByText('Must be a positive integer')
      ).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });

    // Valid: positive integer
    fireEvent.change(numberInput, { target: { value: '123' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.queryByText('Must be a positive integer')
      ).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled();
    });
  });

  it('should allow empty value on optional field with regex', async () => {
    const mockSubmit = vi.fn();
    const schema = createTestSchema(
      {
        required: false,
      },
      {
        regex: {
          pattern: '^[1-9][0-9]{2,}$', // Numbers with 3+ digits, not starting with 0
          message: 'Must be at least 3 digits, not starting with 0',
        },
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

    // Empty should be valid for optional field
    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    // Invalid pattern - only 2 digits
    fireEvent.change(numberInput, { target: { value: '12' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.getByText('Must be at least 3 digits, not starting with 0')
      ).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });

    // Valid pattern
    fireEvent.change(numberInput, { target: { value: '123' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.queryByText('Must be at least 3 digits, not starting with 0')
      ).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled();
    });

    // Clear back to empty - should be valid
    fireEvent.change(numberInput, { target: { value: '' } });
    numberInput.blur();

    await waitFor(() => {
      expect(
        screen.queryByText('Must be at least 3 digits, not starting with 0')
      ).not.toBeInTheDocument();
      expect(submitButton).not.toBeDisabled();
    });
  });
});
