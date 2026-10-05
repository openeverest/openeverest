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

import { useState } from 'react';
import { createTheme } from '@mui/material';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MasonryColumns } from './masonry-columns';

// jsdom has no layout: an item's column is identified by its inline offset.
const columnTexts = () => {
  const columns = new Map<string, string[]>();
  screen.getAllByRole('article').forEach((item) => {
    const offset = item.parentElement?.style.insetInlineStart ?? '';
    columns.set(offset, [
      ...(columns.get(offset) ?? []),
      item.textContent ?? '',
    ]);
  });
  return [...columns.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([, texts]) => texts);
};

const topOf = (name: string) => screen.getByText(name).parentElement?.style.top;

const items = ['a', 'b', 'c', 'd', 'e'].map((name) => (
  <div key={name} role="article">
    {name}
  </div>
));

const mockHeights = (heights: Record<string, number>) =>
  vi
    .spyOn(HTMLElement.prototype, 'offsetHeight', 'get')
    .mockImplementation(function (this: HTMLElement) {
      return heights[this.textContent ?? ''] ?? 0;
    });

let notifyResize = () => {};

class ResizeObserverMock {
  constructor(callback: () => void) {
    notifyResize = callback;
  }
  observe() {}
  unobserve() {}
  disconnect() {}
}

const Counter = ({ name }: { name: string }) => {
  const [clicks, setClicks] = useState(0);
  return (
    <button aria-label={name} onClick={() => setClicks(clicks + 1)}>
      {clicks}
    </button>
  );
};

const mockViewport = (matches: (query: string) => boolean) =>
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: matches(query),
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }));

describe('MasonryColumns', () => {
  beforeEach(() => {
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('puts each item into the shortest column', () => {
    mockHeights({ a: 400, b: 100, c: 150, d: 80, e: 120 });

    render(<MasonryColumns columns={2}>{items}</MasonryColumns>);

    expect(columnTexts()).toEqual([['a'], ['b', 'c', 'd', 'e']]);
  });

  it('treats nearly level columns as equal and fills them left to right', () => {
    mockHeights({ a: 110, b: 100, c: 50 });

    render(<MasonryColumns columns={2}>{items.slice(0, 3)}</MasonryColumns>);

    expect(columnTexts()).toEqual([['a', 'c'], ['b']]);
  });

  it.each([
    ['click', fireEvent.pointerDown],
    ['key press', fireEvent.keyDown],
  ])(
    'keeps items in their column when a sibling grows after a %s',
    (_, interact) => {
      const heights = { a: 100, b: 100, c: 100, d: 100, e: 100 };
      mockHeights(heights);
      render(<MasonryColumns columns={2}>{items}</MasonryColumns>);

      interact(screen.getByText('a'));
      heights.a = 1000;
      notifyResize();

      expect(columnTexts()).toEqual([
        ['a', 'c', 'e'],
        ['b', 'd'],
      ]);
      expect(topOf('c')).toBe('1016px');
    }
  );

  it('closes the gap when an item is removed', async () => {
    mockHeights({ a: 100, b: 100, c: 100, d: 100 });
    const { rerender } = render(
      <MasonryColumns columns={2}>{items.slice(0, 4)}</MasonryColumns>
    );
    fireEvent.pointerDown(screen.getByText('a'));

    rerender(<MasonryColumns columns={2}>{items.slice(1, 4)}</MasonryColumns>);

    await waitFor(() => expect(topOf('c')).toBe('0px'));
    expect(columnTexts()).toEqual([['c'], ['b', 'd']]);
  });

  it('stops observing on unmount', () => {
    const disconnect = vi.spyOn(ResizeObserverMock.prototype, 'disconnect');
    const { unmount } = render(
      <MasonryColumns columns={2}>{items}</MasonryColumns>
    );
    disconnect.mockClear();

    unmount();

    expect(disconnect).toHaveBeenCalled();
  });

  it('re-balances while content is still loading', () => {
    const heights = { a: 100, b: 100, c: 100, d: 100, e: 100 };
    mockHeights(heights);
    render(<MasonryColumns columns={2}>{items}</MasonryColumns>);

    heights.a = 1000;
    notifyResize();

    expect(columnTexts()).toEqual([['a'], ['b', 'c', 'd', 'e']]);
  });

  it('adds a new item to the shortest column without moving the others', async () => {
    mockHeights({ a: 100, b: 400, c: 100, x: 50 });
    const { rerender } = render(
      <MasonryColumns columns={2}>{items.slice(0, 3)}</MasonryColumns>
    );
    fireEvent.pointerDown(screen.getByText('a'));

    rerender(
      <MasonryColumns columns={2}>
        {[
          <div key="x" role="article">
            x
          </div>,
          ...items.slice(0, 3),
        ]}
      </MasonryColumns>
    );

    await waitFor(() =>
      expect(columnTexts()).toEqual([['x', 'a', 'c'], ['b']])
    );
  });

  it('keeps item state when the column count changes', () => {
    const counters = ['a', 'b', 'c', 'd', 'e'].map((name) => (
      <Counter key={name} name={name} />
    ));
    const { rerender } = render(
      <MasonryColumns columns={3}>{counters}</MasonryColumns>
    );

    fireEvent.click(screen.getByRole('button', { name: 'd' }));
    rerender(<MasonryColumns columns={2}>{counters}</MasonryColumns>);

    expect(screen.getByRole('button', { name: 'd' })).toHaveTextContent('1');
  });

  it('uses the largest breakpoint that matches the viewport', () => {
    const { xl } = createTheme().breakpoints.values;
    // Wide enough for lg but not xl.
    mockViewport((query) => !query.includes(`${xl}`));

    render(
      <MasonryColumns columns={{ xs: 1, lg: 2, xl: 3 }}>{items}</MasonryColumns>
    );

    expect(columnTexts()).toHaveLength(2);
  });

  it('falls back to a single column on narrow viewports', () => {
    mockViewport(() => false);

    render(<MasonryColumns columns={{ xs: 1, lg: 3 }}>{items}</MasonryColumns>);

    expect(columnTexts()).toHaveLength(1);
  });

  it('renders at least one column for a zero column count', () => {
    render(<MasonryColumns columns={0}>{items}</MasonryColumns>);

    expect(columnTexts()).toHaveLength(1);
  });

  it('skips empty children', () => {
    render(
      <MasonryColumns columns={2}>
        {items[0]}
        {null}
        {false}
        {items[1]}
      </MasonryColumns>
    );

    expect(columnTexts()).toEqual([['a'], ['b']]);
  });
});
