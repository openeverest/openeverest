import { Button } from '@mui/material';
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
});
