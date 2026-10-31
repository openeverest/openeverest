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
import TruncatedText from './truncated-text';

const CLIENT_WIDTH = 100;
let scrollWidth = CLIENT_WIDTH;

beforeAll(() => {
  Object.defineProperty(HTMLElement.prototype, 'clientWidth', {
    configurable: true,
    get() {
      return CLIENT_WIDTH;
    },
  });
  Object.defineProperty(HTMLElement.prototype, 'scrollWidth', {
    configurable: true,
    get() {
      return scrollWidth;
    },
  });
});

afterAll(() => {
  Reflect.deleteProperty(HTMLElement.prototype, 'clientWidth');
  Reflect.deleteProperty(HTMLElement.prototype, 'scrollWidth');
});

it('should show the full text in a tooltip on hover when it overflows', async () => {
  scrollWidth = CLIENT_WIDTH + 50;
  render(<TruncatedText text="A very long plugin name" />);

  fireEvent.mouseOver(screen.getByTestId('truncated-text'));

  expect(await screen.findByRole('tooltip')).toHaveTextContent(
    'A very long plugin name'
  );
});

it('should not show a tooltip on hover when the text fits', () => {
  scrollWidth = CLIENT_WIDTH;
  render(<TruncatedText text="Short" />);

  fireEvent.mouseOver(screen.getByTestId('truncated-text'));

  expect(screen.queryByRole('tooltip')).toBeNull();
});
