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
import ShowMoreDialog from './show-more-dialog';

it('keeps content hidden until the link opens the dialog, and closes it', () => {
  render(
    <ShowMoreDialog dialogTitle="Rules" dataTestId="rules">
      <span>full content</span>
    </ShowMoreDialog>
  );

  const toggle = screen.getByTestId('rules-toggle');
  expect(toggle).toHaveTextContent('Show more');
  expect(toggle).toHaveAttribute('aria-expanded', 'false');
  expect(screen.queryByText('full content')).toBeNull();

  fireEvent.click(toggle);
  expect(screen.getByText('Rules')).toBeInTheDocument();
  expect(screen.getByText('full content')).toBeInTheDocument();
  expect(toggle).toHaveAttribute('aria-expanded', 'true');

  fireEvent.click(screen.getByTestId('rules-dialog-close'));
  expect(toggle).toHaveAttribute('aria-expanded', 'false');
});
