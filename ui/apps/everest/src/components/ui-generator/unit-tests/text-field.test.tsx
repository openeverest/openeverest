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

import {
  render,
  screen,
  waitFor,
  fireEvent,
  act,
} from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import {
  Component,
  FieldType,
  TextFieldParams,
  TextValidation,
  TopologyUISchemas,
} from '../ui-generator.types';
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
  fieldParams: Partial<TextFieldParams> = {},
  validation?: TextValidation
): TopologyUISchemas => {
  const testText: Extract<Component, { uiType: FieldType.Text }> = {
    uiType: FieldType.Text,
    path: 'spec.testText',
    fieldParams: {
      label: 'Test Text Field',
      ...fieldParams,
    },
  };

  if (validation) {
    testText.validation = validation;
  }

  return {
    testTopology: {
      sections: {
        basicInfo: {
          label: 'Basic Information',
          components: { testText },
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

// Simulate typing a value into a controlled input and trigger blur validation
const typeIntoInput = async (input: HTMLElement, value: string) => {
  await act(async () => {
    fireEvent.change(input, { target: { value } });
    fireEvent.blur(input);
  });
};

describe('UIGenerator - Text Field Basic Rendering', () => {
  it('should render a text input with correct label', async () => {
    const schema = createTestSchema({});

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByLabelText('Test Text Field')).toBeInTheDocument();
  });

  it('should have a data-testid derived from the field path', () => {
    const schema = createTestSchema({});

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    // kebabize('spec.testText') → 'spec.test-text'
    expect(screen.getByTestId('text-input-spec.test-text')).toBeInTheDocument();
  });

  it('should render placeholder text when provided', () => {
    const schema = createTestSchema({ placeholder: 'Enter something' });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByPlaceholderText('Enter something')).toBeInTheDocument();
  });

  it('should render disabled state', () => {
    const schema = createTestSchema({ disabled: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByLabelText('Test Text Field')).toBeDisabled();
  });

  it('should render helperText when provided', () => {
    const schema = createTestSchema({ helperText: 'Some helper text' });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByText('Some helper text')).toBeInTheDocument();
  });
});

describe('UIGenerator - Text Field Multiline', () => {
  it('should render a textarea when multiline=true', () => {
    const schema = createTestSchema({ multiline: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByRole('textbox')).toBeInTheDocument();
    const textarea = screen.getByTestId('text-input-spec.test-text');
    expect(textarea.tagName.toLowerCase()).toBe('textarea');
  });

  it('should accept multi-line input when multiline=true', async () => {
    const schema = createTestSchema({ multiline: true, rows: 4 });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const textarea = screen.getByTestId('text-input-spec.test-text');
    await typeIntoInput(textarea, 'line one\nline two');

    expect(textarea).toHaveValue('line one\nline two');
  });
});

describe('UIGenerator - Text Field Required Validation', () => {
  it('should enable submit when field is optional (default) and empty', async () => {
    const schema = createTestSchema({});

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });

  it('should disable submit when required field is empty', async () => {
    const schema = createTestSchema({}, { required: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  it('should enable submit when required field has a value', async () => {
    const schema = createTestSchema({}, { required: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'hello');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Text Field min Validation', () => {
  it('should disable submit when value is shorter than min', async () => {
    const schema = createTestSchema({}, { min: 5 });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'abc'); // 3 chars < min 5

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  it('should enable submit when value meets min length', async () => {
    const schema = createTestSchema({}, { min: 3 });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'abc'); // exactly 3 chars = min 3

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Text Field max Validation', () => {
  it('should disable submit when value exceeds max length', async () => {
    const schema = createTestSchema({}, { max: 3 });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'abcd'); // 4 chars > max 3

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  it('should enable submit when value is within max length', async () => {
    const schema = createTestSchema({}, { max: 5 });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'abc'); // 3 chars <= max 5

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Text Field length Validation', () => {
  it('should disable submit when value length does not match exact length', async () => {
    const schema = createTestSchema({}, { length: 5 });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'abc'); // 3 ≠ 5

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  it('should enable submit when value has exact required length', async () => {
    const schema = createTestSchema({}, { length: 5 });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'hello'); // exactly 5

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Text Field email Validation', () => {
  it('should disable submit for invalid email format', async () => {
    const schema = createTestSchema({}, { email: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'not-an-email');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  it('should enable submit for a valid email address', async () => {
    const schema = createTestSchema({}, { email: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'user@example.com');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });

  it('should allow empty value when email validation is set but field is optional', async () => {
    const schema = createTestSchema({}, { email: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Text Field url Validation', () => {
  it('should disable submit for an invalid URL', async () => {
    const schema = createTestSchema({}, { url: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'not-a-url');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  it('should enable submit for a valid URL', async () => {
    const schema = createTestSchema({}, { url: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'https://example.com');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Text Field uuid Validation', () => {
  it('should disable submit for an invalid UUID', async () => {
    const schema = createTestSchema({}, { uuid: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'not-a-uuid');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  it('should enable submit for a valid UUID v4', async () => {
    const schema = createTestSchema({}, { uuid: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, '550e8400-e29b-41d4-a716-446655440000');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Text Field trim Transform', () => {
  it('should allow submit when trim is set and value has surrounding whitespace', async () => {
    const onSubmit = vi.fn();
    const schema = createTestSchema({}, { trim: true, required: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={onSubmit}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, '  hello  ');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });

    fireEvent.submit(input.closest('form')!);

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
      const submittedData = onSubmit.mock.calls[0][0];
      expect(submittedData.spec?.testText).toBe('hello');
    });
  });
});

describe('UIGenerator - Text Field toLowerCase Transform', () => {
  it('should transform submitted value to lower case', async () => {
    const onSubmit = vi.fn();
    const schema = createTestSchema({}, { toLowerCase: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={onSubmit}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'HELLO');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });

    fireEvent.submit(input.closest('form')!);

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
      const submittedData = onSubmit.mock.calls[0][0];
      expect(submittedData.spec?.testText).toBe('hello');
    });
  });
});

describe('UIGenerator - Text Field toUpperCase Transform', () => {
  it('should transform submitted value to upper case', async () => {
    const onSubmit = vi.fn();
    const schema = createTestSchema({}, { toUpperCase: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={onSubmit}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'hello');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });

    fireEvent.submit(input.closest('form')!);

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
      const submittedData = onSubmit.mock.calls[0][0];
      expect(submittedData.spec?.testText).toBe('HELLO');
    });
  });
});

describe('UIGenerator - Text Field regex Validation', () => {
  it('should disable submit when value does not match regex pattern', async () => {
    const schema = createTestSchema(
      {},
      { regex: { pattern: '^[a-z]+$', message: 'Only lowercase letters' } }
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'UPPER');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  it('should enable submit when value matches regex pattern', async () => {
    const schema = createTestSchema(
      {},
      { regex: { pattern: '^[a-z]+$', message: 'Only lowercase letters' } }
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');
    await typeIntoInput(input, 'lower');

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});

describe('UIGenerator - Text Field Combined Validations', () => {
  it('should validate required + min together', async () => {
    const schema = createTestSchema({}, { required: true, min: 3 });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');

    // Too short
    await typeIntoInput(input, 'ab');
    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });

    // Meets min
    await typeIntoInput(input, 'abc');
    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });

  it('should validate required + email together', async () => {
    const schema = createTestSchema({}, { required: true, email: true });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="basicInfo"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('Test Text Field');

    await typeIntoInput(input, 'bad-email');
    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });

    await typeIntoInput(input, '');
    await typeIntoInput(input, 'good@example.com');
    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });
});
