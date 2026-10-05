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

import { render, screen, within } from '@testing-library/react';
import MasonryColumns from './masonry-columns';

const columnTexts = () =>
  screen.getAllByTestId('masonry-column').map((column) =>
    within(column)
      .queryAllByRole('article')
      .map((item) => item.textContent)
  );

const items = ['a', 'b', 'c', 'd', 'e'].map((name) => (
  <div key={name} role="article">
    {name}
  </div>
));

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
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // Placement depends only on order, so an item never jumps to another column
  // when a sibling expands.
  it('places items into columns by order, not by height', () => {
    render(<MasonryColumns columns={3}>{items}</MasonryColumns>);

    expect(columnTexts()).toEqual([['a', 'd'], ['b', 'e'], ['c']]);
  });

  it('uses the largest breakpoint that matches the viewport', () => {
    // Wide enough for lg but not xl.
    mockViewport((query) => !query.includes('1536'));

    render(
      <MasonryColumns columns={{ xs: 1, lg: 2, xl: 3 }}>{items}</MasonryColumns>
    );

    expect(columnTexts()).toEqual([
      ['a', 'c', 'e'],
      ['b', 'd'],
    ]);
  });

  it('falls back to a single column on narrow viewports', () => {
    mockViewport(() => false);

    render(<MasonryColumns columns={{ xs: 1, lg: 3 }}>{items}</MasonryColumns>);

    expect(columnTexts()).toEqual([['a', 'b', 'c', 'd', 'e']]);
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
