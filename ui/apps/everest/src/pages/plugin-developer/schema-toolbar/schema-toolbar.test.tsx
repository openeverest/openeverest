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
  waitFor,
  within,
} from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import { SchemaToolbar, SchemaToolbarProps } from './schema-toolbar';
import { Messages } from './schema-toolbar.messages';

// The dialog gates its submit on an async zod resolver, which can lag well
// past the 1s default on a loaded CI runner.
const VALIDATION_WAIT = { timeout: 5000 };

const setup = (props: Partial<SchemaToolbarProps> = {}) => {
  const onSave = vi.fn();
  const onLoad = vi.fn();
  const onDelete = vi.fn();
  const onReset = vi.fn();
  render(
    <TestWrapper>
      <SchemaToolbar
        names={props.names ?? []}
        onSave={props.onSave ?? onSave}
        onLoad={props.onLoad ?? onLoad}
        onDelete={props.onDelete ?? onDelete}
        onReset={props.onReset ?? onReset}
      />
    </TestWrapper>
  );
  return { onSave, onLoad, onDelete, onReset };
};

// The MUI Select renders its options in a portal only once opened.
const selectOption = async (name: string) => {
  fireEvent.mouseDown(screen.getByRole('combobox'));
  await waitFor(() => expect(screen.getByRole('listbox')).toBeInTheDocument());
  const option = screen
    .getAllByRole('option')
    .find((el) => el.textContent === name);
  fireEvent.click(option!);
};

describe('SchemaToolbar', () => {
  it('saves a named schema through the dialog', async () => {
    const { onSave } = setup({ names: [] });

    fireEvent.click(screen.getByRole('button', { name: Messages.save }));

    const dialog = await screen.findByRole('dialog');
    const confirm = within(dialog).getByTestId('form-dialog-save');
    await waitFor(() => expect(confirm).toBeDisabled(), VALIDATION_WAIT);

    fireEvent.change(
      within(dialog).getByLabelText(Messages.saveDialog.nameLabel),
      { target: { value: 'my-schema' } }
    );
    await waitFor(() => expect(confirm).toBeEnabled(), VALIDATION_WAIT);
    fireEvent.click(confirm);

    await waitFor(() => expect(onSave).toHaveBeenCalledWith('my-schema'));
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    );
  });

  it('does not save a whitespace-only name', async () => {
    const { onSave } = setup({ names: [] });

    fireEvent.click(screen.getByRole('button', { name: Messages.save }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.change(
      within(dialog).getByLabelText(Messages.saveDialog.nameLabel),
      { target: { value: '   ' } }
    );

    await waitFor(
      () =>
        expect(within(dialog).getByTestId('form-dialog-save')).toBeDisabled(),
      VALIDATION_WAIT
    );
    expect(onSave).not.toHaveBeenCalled();
  });

  it('saves the trimmed name, not the raw input', async () => {
    const { onSave } = setup({ names: [] });

    fireEvent.click(screen.getByRole('button', { name: Messages.save }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.change(
      within(dialog).getByLabelText(Messages.saveDialog.nameLabel),
      { target: { value: '  spaced  ' } }
    );

    const confirm = within(dialog).getByTestId('form-dialog-save');
    await waitFor(() => expect(confirm).toBeEnabled(), VALIDATION_WAIT);
    fireEvent.click(confirm);

    // The name becomes a storage key, so stray whitespace must not survive.
    await waitFor(() => expect(onSave).toHaveBeenCalledWith('spaced'));
  });

  it('loads the schema chosen in the select', async () => {
    const { onLoad } = setup({ names: ['a', 'b'] });

    await selectOption('b');

    expect(onLoad).toHaveBeenCalledWith('b');
  });

  it('disables Delete until a schema is selected, then deletes it', async () => {
    const { onDelete } = setup({ names: ['a', 'b'] });

    expect(
      screen.getByRole('button', { name: Messages.delete })
    ).toBeDisabled();

    await selectOption('a');

    const deleteBtn = screen.getByRole('button', { name: Messages.delete });
    expect(deleteBtn).toBeEnabled();
    fireEvent.click(deleteBtn);

    expect(onDelete).toHaveBeenCalledWith('a');
  });

  it('resets to the default schema', () => {
    const { onReset } = setup({ names: ['a'] });

    // The tooltip supplies the accessible name; "Reset" is the visible label.
    fireEvent.click(screen.getByRole('button', { name: /reset/i }));

    expect(onReset).toHaveBeenCalledTimes(1);
  });
});
