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

import { Button, Typography } from '@mui/material';
import { TextInput } from '@percona/ui-lib';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { z } from 'zod';
import { FormDialog } from './form-dialog';

enum DataFields {
  name = 'name',
}

const defaultValues = {
  [DataFields.name]: 'Test',
};

const schema = z.object({
  [DataFields.name]: z.string().nonempty(),
});

type DataType = z.infer<typeof schema>;

const Wrapper = () => {
  const [open, setOpen] = useState(false);

  const handleClose = () => {
    setOpen(false);
  };

  const onSubmit = (data: DataType) => {
    alert(data);
  };

  return (
    <div>
      <Button onClick={() => setOpen(true)}>Open Modal</Button>
      <FormDialog
        isOpen={open}
        closeModal={handleClose}
        headerMessage="Add name"
        onSubmit={onSubmit}
        submitMessage="Add"
        schema={schema}
        defaultValues={defaultValues}
      >
        <TextInput name={DataFields.name} label="Name" isRequired />
      </FormDialog>
    </div>
  );
};

describe('FormDialog', () => {
  it('should render correctly', () => {
    render(<Wrapper />);
    const openModalButton = screen.getByText('Open Modal');
    fireEvent.click(openModalButton);
    expect(screen.getByText('Add name')).toBeInTheDocument();
  });

  it('should render with correct fields', () => {
    render(<Wrapper />);
    const openModalButton = screen.getByText('Open Modal');
    fireEvent.click(openModalButton);
    expect(screen.getByText('Name')).toBeInTheDocument();
  });

  it('should close dialog when form is not dirty and there is a click outside', () => {
    const closeModal = vi.fn();

    render(
      <FormDialog
        isOpen
        closeModal={closeModal}
        headerMessage="Test click"
        onSubmit={vi.fn()}
        schema={schema}
        defaultValues={defaultValues}
        submitMessage="Add"
      >
        <TextInput name={DataFields.name} label="Name" isRequired />
      </FormDialog>
    );

    const backdrop = document.body
      .getElementsByClassName('MuiModal-backdrop')
      .item(0);

    expect(backdrop).not.toBeNull();
    fireEvent.click(backdrop!);

    expect(closeModal).toHaveBeenCalled();
  });

  it('should keep dialog open when form is dirty and there is a click outside', async () => {
    const closeModal = vi.fn();

    render(
      <FormDialog
        isOpen
        closeModal={closeModal}
        headerMessage="Test click"
        onSubmit={vi.fn()}
        schema={schema}
        defaultValues={defaultValues}
        submitMessage="Add"
      >
        <TextInput name={DataFields.name} label="Name" isRequired />
      </FormDialog>
    );

    fireEvent.change(screen.getByTestId('text-input-name'), {
      target: { value: 'John' },
    });

    await waitFor(() => screen.getByDisplayValue('John'));

    const backdrop = document.body
      .getElementsByClassName('MuiModal-backdrop')
      .item(0);

    expect(backdrop).not.toBeNull();
    fireEvent.click(backdrop!);

    expect(closeModal).not.toHaveBeenCalled();
  });

  // TEMPORARY: verifies the description slot spacing edit is actually applied.
  describe('description slot spacing', () => {
    const renderWithDescription = () =>
      render(
        <FormDialog
          isOpen
          closeModal={vi.fn()}
          headerMessage="Add rule group"
          description="Create a rule group to control how pods are scheduled onto nodes."
          onSubmit={vi.fn()}
          schema={schema}
          defaultValues={defaultValues}
          submitMessage="Add"
        >
          <Typography variant="sectionHeading">Rule type</Typography>
        </FormDialog>
      );

    it('renders the description and keeps the heading in place', () => {
      renderWithDescription();
      expect(
        screen.getByText(
          'Create a rule group to control how pods are scheduled onto nodes.'
        )
      ).toBeInTheDocument();
      expect(screen.getByText('Rule type')).toBeInTheDocument();
    });

    it('applies the tightened spacing to the description', () => {
      renderWithDescription();
      const description = screen.getByText(
        'Create a rule group to control how pods are scheduled onto nodes.'
      );
      const { marginTop, marginBottom } = getComputedStyle(description);
      expect({ marginTop, marginBottom }).toMatchSnapshot();
    });
  });
});
