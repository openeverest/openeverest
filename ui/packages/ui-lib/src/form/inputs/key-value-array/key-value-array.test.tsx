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

import { fireEvent, render, screen } from '@testing-library/react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { KeyValueArray } from './key-value-array';

const Value = () => {
  const labels = useWatch({ name: 'labels' });
  return <pre data-testid="value">{JSON.stringify(labels)}</pre>;
};

const Form = () => {
  const methods = useForm({
    defaultValues: { labels: [{ key: 'release', value: 'prometheus' }] },
  });
  return (
    <FormProvider {...methods}>
      <KeyValueArray name="labels" label="Labels" />
      <Value />
    </FormProvider>
  );
};

const value = () => JSON.parse(screen.getByTestId('value').textContent ?? '');

describe('KeyValueArray', () => {
  it('adds, edits and removes pairs', () => {
    render(<Form />);

    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
    fireEvent.change(screen.getByRole('textbox', { name: 'Key 2' }), {
      target: { value: 'team' },
    });
    fireEvent.change(screen.getByRole('textbox', { name: 'Value 2' }), {
      target: { value: 'db' },
    });

    expect(value()).toEqual([
      { key: 'release', value: 'prometheus' },
      { key: 'team', value: 'db' },
    ]);

    fireEvent.click(screen.getByRole('button', { name: 'Remove 1' }));

    expect(value()).toEqual([{ key: 'team', value: 'db' }]);
  });
});
