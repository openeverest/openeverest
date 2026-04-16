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

// TODO v2: should be reworked in v2 with implementation of affinity rules

import { DbType } from '@percona/types';
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { AffinityFormDialog } from './affinity-form-dialog';

const Wrapper = ({ dbType }: { dbType: DbType }) => {
  return <AffinityFormDialog isOpen dbType={dbType} />;
};

const selectTypeOption = (optionText: string) => {
  fireEvent.mouseDown(
    within(screen.getByTestId('select-type-button')).getByRole('combobox')
  );
  const option = screen
    .getAllByRole('option')
    .find((o) => o.textContent === optionText);

  expect(option).toBeDefined();
  fireEvent.click(option!);
};

const selectOperatorOption = (optionText: string) => {
  fireEvent.mouseDown(
    within(screen.getByTestId('select-operator-button')).getByRole('combobox')
  );
  const option = screen
    .getAllByRole('option')
    .find((o) => o.textContent === optionText);

  expect(option).toBeDefined();
  fireEvent.click(option!);
};

describe.skip('AffinityFormDialog', () => {
  describe('MongoDB', () => {
    test('show defaults', async () => {
      render(<Wrapper dbType={DbType.Mongo} />);

      await waitFor(() => {
        expect(screen.getByTestId('select-input-component')).toHaveValue(
          'engine'
        );
        expect(screen.getByTestId('select-input-type')).toHaveValue(
          'podAntiAffinity'
        );
        expect(screen.getByTestId('text-input-weight')).toHaveValue(1);
        expect(screen.getByTestId('text-input-topology-key')).toHaveValue(
          'kubernetes.io/hostname'
        );
      });
    }, 10000);
  });

  describe.skip('Node Affinity', () => {
    test('Mandatory key and operator', async () => {
      render(<Wrapper dbType={DbType.Postresql} />);

      await waitFor(() =>
        expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled()
      );
      selectTypeOption('Node affinity');

      await waitFor(() =>
        expect(screen.getByTestId('text-input-key')).toBeInvalid()
      );
      expect(
        screen.queryByTestId('text-input-topology-key')
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('form-dialog-add')).toBeDisabled();
      expect(screen.getByTestId('text-input-key')).toHaveValue('');

      fireEvent.change(screen.getByTestId('text-input-key'), {
        target: { value: 'my-key' },
      });

      await waitFor(() =>
        expect(screen.getByTestId('text-input-key')).not.toBeInvalid()
      );

      expect(screen.getByTestId('select-input-operator')).toHaveValue('');
      expect(screen.getByTestId('select-input-operator')).toBeInvalid();
      expect(screen.getByTestId('form-dialog-add')).toBeDisabled();

      selectOperatorOption('exists');

      await waitFor(() =>
        expect(screen.getByTestId('select-input-operator')).not.toBeInvalid()
      );
      expect(screen.getByTestId('select-input-operator')).toHaveValue('Exists');
      expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled();
    });

    test('Mandatory value if operator is "In" or "NotIn"', async () => {
      render(<Wrapper dbType={DbType.Postresql} />);
      selectTypeOption('Node affinity');
      fireEvent.change(screen.getByTestId('text-input-key'), {
        target: { value: 'my-key' },
      });
      selectOperatorOption('in');

      await waitFor(() =>
        expect(screen.getByTestId('form-dialog-add')).toBeDisabled()
      );

      fireEvent.change(screen.getByTestId('text-input-values'), {
        target: { value: 'val1,val2' },
      });

      await waitFor(() =>
        expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled()
      );

      fireEvent.change(screen.getByTestId('text-input-values'), {
        target: { value: '' },
      });

      await waitFor(() =>
        expect(screen.getByTestId('form-dialog-add')).toBeDisabled()
      );

      selectOperatorOption('not in');

      await waitFor(() =>
        expect(screen.getByTestId('form-dialog-add')).toBeDisabled()
      );

      fireEvent.change(screen.getByTestId('text-input-values'), {
        target: { value: 'val1,val2' },
      });

      await waitFor(() =>
        expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled()
      );
    });
  });

  describe.skip('Pod (Anti-)Affinity', () => {
    ['Pod affinity', 'Pod anti-affinity'].forEach((type) => {
      test(`Mandatory topology key for ${type}`, async () => {
        render(<Wrapper dbType={DbType.Postresql} />);
        selectTypeOption(type);

        await waitFor(() =>
          expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled()
        );
        fireEvent.change(screen.getByTestId('text-input-topology-key'), {
          target: { value: '' },
        });
        await waitFor(() =>
          expect(screen.getByTestId('form-dialog-add')).toBeDisabled()
        );
        fireEvent.change(screen.getByTestId('text-input-topology-key'), {
          target: { value: 'toplogy-key' },
        });
        await waitFor(() =>
          expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled()
        );
      });

      test(`Operator and values mandatory if key not empty for ${type}`, async () => {
        render(<Wrapper dbType={DbType.Postresql} />);
        selectTypeOption(type);
        await waitFor(() =>
          expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled()
        );
        expect(screen.getByTestId('select-input-operator')).toBeDisabled();
        fireEvent.change(screen.getByTestId('text-input-key'), {
          target: { value: 'my-key' },
        });
        await waitFor(() =>
          expect(screen.getByTestId('form-dialog-add')).toBeDisabled()
        );
        expect(screen.getByTestId('select-input-operator')).not.toBeDisabled();
        expect(screen.getByTestId('select-input-operator')).toBeInvalid();
        selectOperatorOption('exists');
        await waitFor(() =>
          expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled()
        );
        selectOperatorOption('in');
        await waitFor(() =>
          expect(screen.getByTestId('form-dialog-add')).toBeDisabled()
        );
        fireEvent.change(screen.getByTestId('text-input-values'), {
          target: { value: 'val1' },
        });
        await waitFor(() =>
          expect(screen.getByTestId('form-dialog-add')).not.toBeDisabled()
        );
      });
    });
  });
});
