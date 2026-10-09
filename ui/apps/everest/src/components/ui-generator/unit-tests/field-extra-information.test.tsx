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

import { render, screen, waitFor, act, fireEvent } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import {
  Component,
  FieldType,
  TopologyUISchemas,
} from '../ui-generator.types';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';

vi.mock('../utils/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

vi.mock('../utils/schema-builder/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

const createSchemaWithInfo = (
  uiType: FieldType.Text | FieldType.Number | FieldType.Select,
  info?: string
): TopologyUISchemas => {
  const baseComponent: Partial<Component> = {
    path: 'spec.testField',
    fieldParams: {
      label: 'Test Field',
      ...(info !== undefined ? { info } : {}),
    },
  };

  let component: Component;

  switch (uiType) {
    case FieldType.Number:
      component = {
        ...(baseComponent as Omit<Component, 'uiType'>),
        uiType: FieldType.Number,
        fieldParams: { label: 'Test Field', ...(info ? { info } : {}) },
      } as Component;
      break;
    case FieldType.Select:
      component = {
        ...(baseComponent as Omit<Component, 'uiType'>),
        uiType: FieldType.Select,
        fieldParams: {
          label: 'Test Field',
          options: [{ label: 'Option A', value: 'a' }],
          ...(info ? { info } : {}),
        },
      } as Component;
      break;
    default:
      component = {
        ...(baseComponent as Omit<Component, 'uiType'>),
        uiType: FieldType.Text,
        fieldParams: { label: 'Test Field', ...(info ? { info } : {}) },
      } as Component;
  }

  return {
    testTopology: {
      sections: {
        basicInfo: {
          label: 'Basic Information',
          components: { testField: component },
        },
      },
      sectionsOrder: ['basicInfo'],
    },
  };
};

interface FormWrapperProps {
  children: React.ReactNode;
  schema: TopologyUISchemas;
}

const FormWrapper = ({ children, schema }: FormWrapperProps) => {
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
      <form>{children}</form>
    </FormProvider>
  );
};

const renderField = (
  uiType: FieldType.Text | FieldType.Number | FieldType.Select,
  info?: string
) => {
  const schema = createSchemaWithInfo(uiType, info);
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
};

describe('UIGenerator - Field Extra Information (info property)', () => {
  describe('Text field', () => {
    it('does not render the info button when info is not set', () => {
      renderField(FieldType.Text);
      expect(
        screen.queryByTestId('field-info-button')
      ).not.toBeInTheDocument();
    });

    it('renders the info button when info is set', () => {
      renderField(FieldType.Text, 'This is extra context for the text field.');
      expect(screen.getByTestId('field-info-button')).toBeInTheDocument();
    });

    it('info button has correct aria-label for accessibility', () => {
      renderField(FieldType.Text, 'Accessible info text');
      expect(
        screen.getByRole('button', { name: 'Field information' })
      ).toBeInTheDocument();
    });

    it('shows the info text in a tooltip on hover', async () => {
      const infoText = 'Hover tooltip content for text field';
      renderField(FieldType.Text, infoText);

      const infoButton = screen.getByTestId('field-info-button');
      await act(async () => {
        fireEvent.mouseOver(infoButton);
      });

      await waitFor(() => {
        expect(screen.getByRole('tooltip')).toHaveTextContent(infoText);
      });
    });

    it('still renders the field label when info is set', () => {
      renderField(FieldType.Text, 'Some info');
      expect(screen.getByLabelText('Test Field')).toBeInTheDocument();
    });
  });

  describe('Number field', () => {
    it('does not render the info button when info is not set', () => {
      renderField(FieldType.Number);
      expect(
        screen.queryByTestId('field-info-button')
      ).not.toBeInTheDocument();
    });

    it('renders the info button when info is set', () => {
      renderField(FieldType.Number, 'Must be between 1 and 100.');
      expect(screen.getByTestId('field-info-button')).toBeInTheDocument();
    });

    it('shows the info text in a tooltip on hover', async () => {
      const infoText = 'Number field tooltip content';
      renderField(FieldType.Number, infoText);

      const infoButton = screen.getByTestId('field-info-button');
      await act(async () => {
        fireEvent.mouseOver(infoButton);
      });

      await waitFor(() => {
        expect(screen.getByRole('tooltip')).toHaveTextContent(infoText);
      });
    });
  });

  describe('Select field', () => {
    it('does not render the info button when info is not set', () => {
      renderField(FieldType.Select);
      expect(
        screen.queryByTestId('field-info-button')
      ).not.toBeInTheDocument();
    });

    it('renders the info button when info is set', () => {
      renderField(FieldType.Select, 'Choose the appropriate storage class.');
      expect(screen.getByTestId('field-info-button')).toBeInTheDocument();
    });

    it('shows the info text in a tooltip on hover', async () => {
      const infoText = 'Select field tooltip content';
      renderField(FieldType.Select, infoText);

      const infoButton = screen.getByTestId('field-info-button');
      await act(async () => {
        fireEvent.mouseOver(infoButton);
      });

      await waitFor(() => {
        expect(screen.getByRole('tooltip')).toHaveTextContent(infoText);
      });
    });
  });

  describe('info does not interfere with other field features', () => {
    it('renders both info button and disabled state correctly', () => {
      const schema = createSchemaWithInfo(FieldType.Text, 'Some info');
      const textComponent = schema.testTopology!.sections.basicInfo
        .components.testField as Component;
      (textComponent.fieldParams as { disabled?: boolean }).disabled = true;

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

      expect(screen.getByTestId('field-info-button')).toBeInTheDocument();
      expect(screen.getByLabelText('Test Field')).toBeDisabled();
    });

    it('renders both info and tooltip wrappers when both are set', () => {
      const schema = createSchemaWithInfo(FieldType.Text, 'Info text');
      const textComponent = schema.testTopology!.sections.basicInfo
        .components.testField as Component;
      (textComponent.fieldParams as { tooltip?: string }).tooltip =
        'Tooltip text';

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

      expect(screen.getByTestId('field-info-button')).toBeInTheDocument();
      expect(screen.getByTestId('field-tooltip')).toBeInTheDocument();
    });
  });
});
