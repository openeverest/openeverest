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

import { ReactNode, useState } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MasonryColumns } from './masonry-columns';

// With a 16px gap, 400px columns fit 1 / 2 / 3 times in 400 / 832 / 1248px.
const masonry = (children: ReactNode, maxColumns?: number) => (
  <MasonryColumns
    minColumnWidth={400}
    maxColumns={maxColumns}
    dataTestId="masonry"
  >
    {children}
  </MasonryColumns>
);

const itemsOf = () =>
  Array.from(screen.getByTestId('masonry').children).filter(
    (child): child is HTMLElement => child instanceof HTMLElement
  );

// jsdom has no layout: an item's column is identified by its inline offset.
const columnTexts = () => {
  const columns = new Map<string, string[]>();
  itemsOf().forEach((item) => {
    const offset = item.style.insetInlineStart;
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
  <div key={name}>{name}</div>
));

const mockHeights = (heights: Record<string, number>) =>
  vi
    .spyOn(HTMLElement.prototype, 'offsetHeight', 'get')
    .mockImplementation(function (this: HTMLElement) {
      return heights[this.textContent ?? ''] ?? 0;
    });

let containerWidth = 832;
let resizeCallbacks: (() => void)[] = [];
const notifyResize = () => resizeCallbacks.forEach((callback) => callback());

class ResizeObserverMock {
  constructor(callback: () => void) {
    resizeCallbacks.push(callback);
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

describe('MasonryColumns', () => {
  beforeEach(() => {
    containerWidth = 832;
    resizeCallbacks = [];
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(
      () => containerWidth
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('puts each item into the shortest column', () => {
    mockHeights({ a: 400, b: 100, c: 150, d: 80, e: 120 });

    render(masonry(items));

    expect(columnTexts()).toEqual([['a'], ['b', 'c', 'd', 'e']]);
  });

  it('treats nearly level columns as equal and fills them left to right', () => {
    mockHeights({ a: 110, b: 100, c: 50 });

    render(masonry(items.slice(0, 3)));

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
      render(masonry(items));

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

  it('re-balances while content is still loading', () => {
    const heights = { a: 100, b: 100, c: 100, d: 100, e: 100 };
    mockHeights(heights);
    render(masonry(items));

    heights.a = 1000;
    notifyResize();

    expect(columnTexts()).toEqual([['a'], ['b', 'c', 'd', 'e']]);
  });

  it('adds a new item to the shortest column without moving the others', async () => {
    mockHeights({ a: 100, b: 400, c: 100, x: 50 });
    const { rerender } = render(masonry(items.slice(0, 3)));
    fireEvent.pointerDown(screen.getByText('a'));

    rerender(masonry([<div key="x">x</div>, ...items.slice(0, 3)]));

    await waitFor(() =>
      expect(columnTexts()).toEqual([['x', 'a', 'c'], ['b']])
    );
  });

  it('closes the gap when an item is removed', async () => {
    mockHeights({ a: 100, b: 100, c: 100, d: 100 });
    const { rerender } = render(masonry(items.slice(0, 4)));
    fireEvent.pointerDown(screen.getByText('a'));

    rerender(masonry(items.slice(1, 4)));

    await waitFor(() => expect(topOf('c')).toBe('0px'));
    expect(columnTexts()).toEqual([['c'], ['b', 'd']]);
  });

  it.each([
    [400, 1],
    [832, 2],
    [1248, 3],
  ])('fits a %ipx container with %i columns', (width, columns) => {
    containerWidth = width;

    render(masonry(items));

    expect(columnTexts()).toHaveLength(columns);
  });

  it('never exceeds maxColumns, and keeps at least one column', () => {
    containerWidth = 1248;

    const { rerender } = render(masonry(items, 2));
    expect(columnTexts()).toHaveLength(2);

    rerender(masonry(items, 0));
    expect(columnTexts()).toHaveLength(1);
  });

  it('re-flows on container resize without remounting items', async () => {
    containerWidth = 1248;
    render(
      masonry(
        ['a', 'b', 'c', 'd', 'e'].map((name) => (
          <Counter key={name} name={name} />
        ))
      )
    );
    fireEvent.click(screen.getByRole('button', { name: 'd' }));

    containerWidth = 832;
    notifyResize();

    await waitFor(() => expect(columnTexts()).toHaveLength(2));
    expect(screen.getByRole('button', { name: 'd' })).toHaveTextContent('1');
  });

  it('stops observing on unmount', () => {
    const disconnect = vi.spyOn(ResizeObserverMock.prototype, 'disconnect');
    const { unmount } = render(masonry(items));
    disconnect.mockClear();

    unmount();

    expect(disconnect).toHaveBeenCalled();
  });

  it('skips empty children', () => {
    render(masonry([items[0], null, false, items[1]]));

    expect(columnTexts()).toEqual([['a'], ['b']]);
  });
});
