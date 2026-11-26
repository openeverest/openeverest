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
import KeyValueSummary from './key-value-summary';

const entries = [
  { key: 'service.beta.kubernetes.io/aws-load-balancer-type', value: 'nlb' },
  { key: 'team', value: 'db' },
];

it('keeps the entries behind a count chip until it opens the dialog', () => {
  render(
    <KeyValueSummary entries={entries} label="Annotations" dataTestId="anno" />
  );

  const toggle = screen.getByTestId('anno-toggle');
  expect(toggle).toHaveTextContent('2 entries');
  expect(toggle).toHaveAttribute('aria-expanded', 'false');
  expect(screen.queryByText('nlb')).toBeNull();

  fireEvent.click(toggle);
  expect(screen.getByRole('dialog')).toHaveAccessibleName('Annotations');
  expect(
    screen.getByText('service.beta.kubernetes.io/aws-load-balancer-type')
  ).toBeInTheDocument();
  expect(screen.getByText('nlb')).toBeInTheDocument();
  expect(screen.getByText('db')).toBeInTheDocument();

  fireEvent.click(screen.getByTestId('anno-dialog-close'));
  expect(toggle).toHaveAttribute('aria-expanded', 'false');
});

it('counts a single entry in the singular', () => {
  render(<KeyValueSummary entries={[entries[1]]} label="Annotations" />);

  expect(screen.getByTestId('key-value-summary-toggle')).toHaveTextContent(
    '1 entry'
  );
});

it('copies a row as key=value', () => {
  const writeText = vi.fn();
  Object.assign(navigator, { clipboard: { writeText } });

  render(<KeyValueSummary entries={entries} label="Annotations" />);
  fireEvent.click(screen.getByTestId('key-value-summary-toggle'));
  fireEvent.click(screen.getByLabelText('Copy team'));

  expect(writeText).toHaveBeenCalledWith('team=db');
});

it('offers no copy for a row that shows a mask', () => {
  render(
    <KeyValueSummary
      entries={[
        { key: 'TOKEN', value: '***************', masked: true },
        { key: 'REGION', value: 'eu-west-1' },
      ]}
      label="Environment"
    />
  );
  fireEvent.click(screen.getByTestId('key-value-summary-toggle'));

  expect(screen.queryByLabelText('Copy TOKEN')).toBeNull();
  expect(screen.getByLabelText('Copy REGION')).toBeInTheDocument();
});
