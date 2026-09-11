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

import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  render,
  screen,
  fireEvent,
  waitFor,
  within,
} from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import { UIGeneratorBuilder } from './ui-generator-builder';
import { formatYamlText } from './utils/yaml-json-converter';
import { Messages as OutputPanelMessages } from './output-panel/output-panel.messages';
import { Messages as SchemaToolbarMessages } from './schema-toolbar/schema-toolbar.messages';

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

// Stubbed so onGenerateOutput can be driven without a real form submission.
vi.mock('./dynamic-form-preview/dynamic-form-preview', () => ({
  DynamicForm: ({
    namespace,
    onGenerateOutput,
  }: {
    namespace?: string;
    onGenerateOutput?: (payload: Record<string, unknown>) => void;
  }) => (
    <div data-testid="preview" data-namespace={namespace}>
      <button onClick={() => onGenerateOutput?.({ engine: { type: 'pxc' } })}>
        emit-output
      </button>
    </div>
  ),
}));

const getEditor = () => screen.getByLabelText('yaml') as HTMLTextAreaElement;

const renderBuilder = () =>
  render(
    <TestWrapper>
      <UIGeneratorBuilder />
    </TestWrapper>
  );

describe('UIGeneratorBuilder', () => {
  // The page reads its draft from localStorage, so start each test clean.
  beforeEach(() => localStorage.clear());

  it('renders the preview for a valid schema', () => {
    renderBuilder();

    expect(screen.getByTestId('preview')).toBeInTheDocument();
    expect(screen.queryByTestId('error-count')).not.toBeInTheDocument();
  });

  it('hides the preview and surfaces errors for an invalid schema', () => {
    renderBuilder();

    fireEvent.change(getEditor(), { target: { value: 'just a string' } });

    expect(screen.queryByTestId('preview')).not.toBeInTheDocument();
    expect(screen.getByTestId('error-count')).toBeInTheDocument();
  });

  it('defaults the preview namespace to the first available one', () => {
    renderBuilder();

    expect(screen.getByTestId('preview')).toHaveAttribute(
      'data-namespace',
      'ns-a'
    );
  });

  it('passes a newly selected namespace down to the preview', async () => {
    renderBuilder();

    // Two selects exist now (saved schemas + namespace); target by label.
    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: /preview namespace/i })
    );
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
    renderBuilder();

    fireEvent.click(screen.getByRole('button', { name: 'Format YAML' }));

    expect(getEditor().value).toBe(formatYamlText(VALID_YAML));
  });

  it('restores the autosaved draft on mount', () => {
    const draft = 'restored: true\n';
    localStorage.setItem('everest.playground.draft', JSON.stringify(draft));

    renderBuilder();

    expect(getEditor().value).toBe(draft);
  });

  it('shows the generated payload in the Output tab after submit', () => {
    renderBuilder();

    fireEvent.click(screen.getByRole('button', { name: 'emit-output' }));

    const output = screen.getByTestId('output-json');
    expect(output.textContent).toContain('engine');
    expect(output.textContent).toContain('pxc');
  });

  it('keeps the form mounted when the Output tab is shown', () => {
    renderBuilder();
    const preview = screen.getByTestId('preview');

    fireEvent.click(screen.getByRole('button', { name: 'emit-output' }));

    // Unmounting would discard everything already typed into the form.
    expect(screen.getByTestId('preview')).toBe(preview);
  });

  it('clears a generated payload when the schema changes', () => {
    renderBuilder();

    fireEvent.click(screen.getByRole('button', { name: 'emit-output' }));
    expect(screen.getByTestId('output-json')).toBeInTheDocument();

    // The old payload no longer describes what's on screen.
    fireEvent.change(getEditor(), { target: { value: VALID_YAML + '\n' } });

    expect(screen.queryByTestId('output-json')).not.toBeInTheDocument();
    expect(
      screen.getByText(OutputPanelMessages.emptyState)
    ).toBeInTheDocument();
  });

  it('clears a stale payload when the loaded schema matches the editor', async () => {
    // Same YAML as the editor already holds: the load still remounts the form,
    // so a payload from the old values must not survive it.
    localStorage.setItem(
      'everest.playground.saved',
      JSON.stringify({ same: VALID_YAML })
    );

    renderBuilder();

    fireEvent.click(screen.getByRole('button', { name: 'emit-output' }));
    expect(screen.getByTestId('output-json')).toBeInTheDocument();

    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: /saved schemas/i })
    );
    await waitFor(() =>
      expect(screen.getByRole('listbox')).toBeInTheDocument()
    );
    fireEvent.click(
      screen.getAllByRole('option').find((el) => el.textContent === 'same')!
    );

    expect(screen.queryByTestId('output-json')).not.toBeInTheDocument();
  });

  it('saves the current schema under a name', async () => {
    renderBuilder();

    fireEvent.click(
      screen.getByRole('button', { name: SchemaToolbarMessages.save })
    );
    const dialog = screen.getByRole('dialog');
    fireEvent.change(
      within(dialog).getByLabelText(SchemaToolbarMessages.saveDialog.nameLabel),
      { target: { value: 'mine' } }
    );
    const confirm = within(dialog).getByTestId('form-dialog-save');
    await waitFor(() => expect(confirm).toBeEnabled(), { timeout: 5000 });
    fireEvent.click(confirm);

    // The builder pairs the chosen name with the current editor YAML.
    await waitFor(() =>
      expect(
        JSON.parse(localStorage.getItem('everest.playground.saved')!)
      ).toEqual({ mine: VALID_YAML })
    );
  });

  it('loads a saved schema into the editor', async () => {
    const seeded = 'seeded: schema\n';
    localStorage.setItem(
      'everest.playground.saved',
      JSON.stringify({ seeded })
    );

    renderBuilder();

    // Move the editor away from the seed so the load is observable.
    fireEvent.change(getEditor(), { target: { value: 'other: value' } });
    expect(getEditor().value).toBe('other: value');

    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: /saved schemas/i })
    );
    await waitFor(() =>
      expect(screen.getByRole('listbox')).toBeInTheDocument()
    );
    fireEvent.click(
      screen.getAllByRole('option').find((el) => el.textContent === 'seeded')!
    );

    expect(getEditor().value).toBe(seeded);
  });
});
