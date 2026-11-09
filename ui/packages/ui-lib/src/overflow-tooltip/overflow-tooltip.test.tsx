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

import { useRef } from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { OverflowTooltip } from './overflow-tooltip';

const CLIENT_WIDTH = 100;
let scrollWidth = CLIENT_WIDTH;

const Row = ({ text }: { text: string }) => {
  const labelRef = useRef<HTMLSpanElement>(null);

  return (
    <OverflowTooltip title={text} measureRef={labelRef}>
      <a href="/plugin">
        <span ref={labelRef}>{text}</span>
      </a>
    </OverflowTooltip>
  );
};

const flushTooltipDelay = () =>
  act(() => {
    vi.runAllTimers();
  });

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

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

it('should show the full text when hovering anywhere on the anchor and the label overflows', () => {
  scrollWidth = CLIENT_WIDTH + 50;
  render(<Row text="A very long plugin name" />);

  fireEvent.mouseOver(screen.getByRole('link'));
  flushTooltipDelay();

  expect(screen.getByRole('tooltip')).toHaveTextContent(
    'A very long plugin name'
  );
});

it('should not show a tooltip on hover when the label fits', () => {
  scrollWidth = CLIENT_WIDTH;
  render(<Row text="Short" />);

  fireEvent.mouseOver(screen.getByRole('link'));
  flushTooltipDelay();

  expect(screen.queryByRole('tooltip')).toBeNull();
});

it('should show the full text when the anchor receives keyboard focus and the label overflows', () => {
  scrollWidth = CLIENT_WIDTH + 50;
  // MUI opens tooltips on focus only when the target matches :focus-visible,
  // which jsdom doesn't report yet while the focus event is being dispatched.
  const nativeMatches = Element.prototype.matches;
  vi.spyOn(Element.prototype, 'matches').mockImplementation(function (
    this: Element,
    selector: string
  ) {
    return selector === ':focus-visible' || nativeMatches.call(this, selector);
  });
  render(<Row text="A very long plugin name" />);

  act(() => {
    screen.getByRole('link').focus();
  });
  flushTooltipDelay();

  expect(screen.getByRole('tooltip')).toHaveTextContent(
    'A very long plugin name'
  );
});
