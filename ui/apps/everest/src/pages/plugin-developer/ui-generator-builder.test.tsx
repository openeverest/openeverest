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
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import { UIGeneratorBuilder } from './ui-generator-builder';
import { formatYamlText } from './utils/yaml-json-converter';

// Hoisted so the vi.mock factory below (also hoisted) can read it.
const { VALID_YAML } = vi.hoisted(() => ({
  VALID_YAML: [
    'standalone:',
    '  sections:',
    '    basics:',
    '      components:',
    '        nodes:',
    '          uiType: number',
    '          path: x',
    '',
  ].join('\n'),
}));

// Pin the editor's seed schema so the preview is deterministic.
vi.mock('components/ui-generator/ui-generator.mock.yaml?raw', () => ({
  default: VALID_YAML,
}));

vi.mock('hooks/api/namespaces', () => ({
  useNamespaces: () => ({ data: ['ns-a', 'ns-b'], isLoading: false }),
}));

// Stub the editor as a plain textarea to drive the YAML directly.
vi.mock('./yaml-editor-panel/yaml-editor-panel', () => ({
  YamlEditorPanel: ({
    yamlText,
    diagnostics,
    onChange,
    onFormat,
  }: {
    yamlText: string;
    diagnostics: { severity: string }[];
    onChange: (v: string) => void;
    onFormat: () => void;
  }) => {
    const errors = diagnostics.filter((d) => d.severity === 'error').length;
    return (
      <div>
        <textarea
          aria-label="yaml"
          value={yamlText}
          onChange={(e) => onChange(e.target.value)}
        />
        {errors > 0 && <span data-testid="error-count">{errors}</span>}
        <button onClick={onFormat}>Format YAML</button>
      </div>
    );
  },
}));

// Stub the preview form to assert it mounts with the right namespace.
vi.mock('./dynamic-form-preview/dynamic-form-preview', () => ({
  DynamicForm: ({ namespace }: { namespace?: string }) => (
    <div data-testid="preview" data-namespace={namespace} />
  ),
}));

const getEditor = () => screen.getByLabelText('yaml') as HTMLTextAreaElement;

describe('UIGeneratorBuilder', () => {
  it('renders the preview for a valid schema', () => {
    render(
      <TestWrapper>
        <UIGeneratorBuilder />
      </TestWrapper>
    );

    expect(screen.getByTestId('preview')).toBeInTheDocument();
    expect(screen.queryByTestId('error-count')).not.toBeInTheDocument();
  });

  it('hides the preview and surfaces errors for an invalid schema', () => {
    render(
      <TestWrapper>
        <UIGeneratorBuilder />
      </TestWrapper>
    );

    fireEvent.change(getEditor(), { target: { value: 'just a string' } });

    expect(screen.queryByTestId('preview')).not.toBeInTheDocument();
    expect(screen.getByTestId('error-count')).toBeInTheDocument();
  });

  it('defaults the preview namespace to the first available one', () => {
    render(
      <TestWrapper>
        <UIGeneratorBuilder />
      </TestWrapper>
    );

    expect(screen.getByTestId('preview')).toHaveAttribute(
      'data-namespace',
      'ns-a'
    );
  });

  it('passes a newly selected namespace down to the preview', async () => {
    render(
      <TestWrapper>
        <UIGeneratorBuilder />
      </TestWrapper>
    );

    // The MUI Select renders its options in a portal only once opened.
    fireEvent.mouseDown(screen.getByRole('combobox'));
    await waitFor(() =>
      expect(screen.getByRole('listbox')).toBeInTheDocument()
    );
    const option = screen
      .getAllByRole('option')
      .find((el) => el.textContent === 'ns-b');
    fireEvent.click(option!);

    expect(screen.getByTestId('preview')).toHaveAttribute(
      'data-namespace',
      'ns-b'
    );
  });

  it('reformats the YAML when Format is pressed', () => {
    render(
      <TestWrapper>
        <UIGeneratorBuilder />
      </TestWrapper>
    );

    fireEvent.click(screen.getByRole('button', { name: 'Format YAML' }));

    expect(getEditor().value).toBe(formatYamlText(VALID_YAML));
  });
});
