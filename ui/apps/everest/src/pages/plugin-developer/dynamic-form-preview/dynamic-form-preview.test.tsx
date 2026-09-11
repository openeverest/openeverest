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

import { describe, it, expect, vi } from 'vitest';
import {
  render,
  screen,
  fireEvent,
  act,
  waitFor,
} from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import {
  Component,
  FieldType,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { postprocessSchemaData } from 'components/ui-generator/utils/postprocess/postprocess-schema';
import { DynamicForm } from './dynamic-form-preview';

// One section means a single step, so Submit is shown immediately.
const SELECTED_TOPOLOGY = 'testTopology';

const buildSchema = (): TopologyUISchemas => {
  const testText: Extract<Component, { uiType: FieldType.Text }> = {
    uiType: FieldType.Text,
    path: 'spec.testText',
    fieldParams: { label: 'Test Text Field' },
    validation: { required: true },
  };

  return {
    [SELECTED_TOPOLOGY]: {
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

const buildMultiSectionSchema = (names: string[]): TopologyUISchemas => ({
  [SELECTED_TOPOLOGY]: {
    sections: Object.fromEntries(
      names.map((name) => [
        name,
        {
          label: `Section ${name}`,
          components: {
            [`${name}Field`]: {
              uiType: FieldType.Text,
              path: `spec.${name}`,
              fieldParams: { label: `${name} field` },
            } as Component,
          },
        },
      ])
    ),
    sectionsOrder: names,
  },
});

describe('DynamicForm - onGenerateOutput', () => {
  it('emits the post-processed payload when Submit is clicked', async () => {
    const schema = buildSchema();
    const onGenerateOutput = vi.fn();

    render(
      <TestWrapper>
        <DynamicForm schema={schema} onGenerateOutput={onGenerateOutput} />
      </TestWrapper>
    );

    // Fill the required field so formState.isValid becomes true and Submit enables.
    const input = screen.getByLabelText('Test Text Field');
    await act(async () => {
      fireEvent.change(input, { target: { value: 'hello' } });
      fireEvent.blur(input);
    });

    const submit = screen.getByTestId('db-wizard-submit-button');
    await waitFor(() => expect(submit).not.toBeDisabled());

    fireEvent.click(submit);

    const expectedValues = {
      topology: { type: SELECTED_TOPOLOGY },
      spec: { testText: 'hello' },
    };
    const expectedPayload = postprocessSchemaData(expectedValues, {
      schema,
      selectedTopology: SELECTED_TOPOLOGY,
    });

    expect(onGenerateOutput).toHaveBeenCalledTimes(1);
    expect(onGenerateOutput).toHaveBeenCalledWith(expectedPayload);
  });
});

describe('DynamicForm - schema swap', () => {
  it('keeps rendering a section when the new schema has fewer steps', () => {
    const wide = buildMultiSectionSchema(['alpha', 'beta', 'gamma']);
    const narrow = buildMultiSectionSchema(['solo']);

    const { rerender } = render(
      <TestWrapper>
        <DynamicForm schema={wide} />
      </TestWrapper>
    );

    // Walk to the last step of the wider schema.
    fireEvent.click(screen.getByTestId('db-wizard-continue-button'));
    fireEvent.click(screen.getByTestId('db-wizard-continue-button'));
    expect(screen.getByLabelText('gamma field')).toBeInTheDocument();

    // Loading a smaller schema must not leave the step index out of range.
    rerender(
      <TestWrapper>
        <DynamicForm schema={narrow} />
      </TestWrapper>
    );

    expect(screen.getByLabelText('solo field')).toBeInTheDocument();
  });
});
