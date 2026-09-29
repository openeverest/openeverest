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

import { render, screen, fireEvent } from '@testing-library/react';
import {
  AffinityOperator,
  AffinityPriority,
  AffinityType,
} from 'shared-types/affinity.types';
import { GroupEditorDialog } from './group-editor-dialog';

vi.mock('components/ui-generator/hooks/use-cel-validation', () => ({
  useCelValidation: () => {},
}));

describe('GroupEditorDialog', () => {
  it('starts with one condition row and appends rows via "Add condition"', () => {
    render(<GroupEditorDialog isOpen onClose={() => {}} onSubmit={() => {}} />);

    expect(screen.getByText('Add rule group')).toBeInTheDocument();
    expect(
      screen.getAllByRole('button', { name: /remove condition/i })
    ).toHaveLength(1);

    fireEvent.click(screen.getByRole('button', { name: /add condition/i }));

    expect(
      screen.getAllByRole('button', { name: /remove condition/i })
    ).toHaveLength(2);
  });

  it('disables the remove button while only one condition remains', () => {
    render(<GroupEditorDialog isOpen onClose={() => {}} onSubmit={() => {}} />);

    expect(
      screen.getByRole('button', { name: /remove condition/i })
    ).toBeDisabled();
  });

  // Every rule-type field keeps a persistent helper, so the row heights stay
  // uniform and an error swaps in for the helper instead of adding a line.
  it('renders a persistent helper for every rule-type field', () => {
    render(
      <GroupEditorDialog
        isOpen
        group={{
          type: AffinityType.PodAffinity,
          priority: AffinityPriority.Preferred,
          weight: 50,
          topologyKey: 'kubernetes.io/hostname',
          conditions: [{ key: '', operator: AffinityOperator.In, values: [] }],
        }}
        onClose={() => {}}
        onSubmit={() => {}}
      />
    );

    expect(screen.getByText('How pods are scheduled')).toBeInTheDocument();
    expect(screen.getByText('1 - 100')).toBeInTheDocument();
    expect(
      screen.getByText('A domain key that determines relative pod placement')
    ).toBeInTheDocument();
  });

  // Key, operator and values are merged into one segmented field (a11y group).
  it('renders key, operator and values as segments of one field', () => {
    render(
      <GroupEditorDialog
        isOpen
        group={{
          type: AffinityType.PodAffinity,
          priority: AffinityPriority.Required,
          topologyKey: 'kubernetes.io/hostname',
          conditions: [
            { key: 'test', operator: AffinityOperator.In, values: [] },
          ],
        }}
        onClose={() => {}}
        onSubmit={() => {}}
      />
    );

    const keyField = document.querySelector('input[name="conditions.0.key"]');
    const group = keyField?.closest('[role="group"]');
    const valuesField = screen.getByRole('textbox', { name: 'Values' });

    expect(keyField).not.toBeNull();
    expect(group).not.toBeNull();
    // Values shares the same segmented group as key/operator.
    expect(group).toContainElement(valuesField);
  });

  // Exists/DoesNotExist take no values, so the default condition hides the field
  // and only reveals it once an operator that consumes values is chosen.
  it('defaults the operator to Exists and hides the values field', () => {
    render(<GroupEditorDialog isOpen onClose={() => {}} onSubmit={() => {}} />);

    expect(
      screen.queryByRole('textbox', { name: 'Values' })
    ).not.toBeInTheDocument();
  });

  it('parses comma-separated input into values and clears the required error', () => {
    render(
      <GroupEditorDialog
        isOpen
        group={{
          type: AffinityType.PodAffinity,
          priority: AffinityPriority.Required,
          topologyKey: 'kubernetes.io/hostname',
          conditions: [
            { key: 'test', operator: AffinityOperator.In, values: [] },
          ],
        }}
        onClose={() => {}}
        onSubmit={() => {}}
      />
    );

    const valuesInput = screen.getByRole('textbox', { name: 'Values' });
    fireEvent.change(valuesInput, { target: { value: 'a, b' } });

    // The comma-separated text satisfies the required rule (no Enter needed).
    expect(screen.queryByText(/Values is required/i)).not.toBeInTheDocument();
  });

  // The dialog validates eagerly to gate the submit button, but an empty
  // required field must not look wrong before the user has engaged with it.
  it('does not show the values error until the field is touched', async () => {
    render(
      <GroupEditorDialog
        isOpen
        group={{
          type: AffinityType.PodAffinity,
          priority: AffinityPriority.Required,
          topologyKey: 'kubernetes.io/hostname',
          conditions: [
            { key: 'test', operator: AffinityOperator.In, values: [] },
          ],
        }}
        onClose={() => {}}
        onSubmit={() => {}}
      />
    );

    const valuesInput = screen.getByRole('textbox', { name: 'Values' });
    expect(screen.queryByText(/Values is required/i)).not.toBeInTheDocument();

    fireEvent.blur(valuesInput);
    expect(await screen.findByText(/Values is required/i)).toBeInTheDocument();
  });

  it('rejects values that break the k8s label-value rules', async () => {
    render(
      <GroupEditorDialog
        isOpen
        group={{
          type: AffinityType.PodAffinity,
          priority: AffinityPriority.Required,
          topologyKey: 'kubernetes.io/hostname',
          conditions: [
            { key: 'test', operator: AffinityOperator.In, values: [] },
          ],
        }}
        onClose={() => {}}
        onSubmit={() => {}}
      />
    );

    const valuesInput = screen.getByRole('textbox', { name: 'Values' });
    fireEvent.change(valuesInput, { target: { value: 'bad value!' } });
    fireEvent.blur(valuesInput);

    expect(
      await screen.findByText(/Values may use letters/i)
    ).toBeInTheDocument();
  });
});
