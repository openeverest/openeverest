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

import { MasonryLayoutOptions } from './masonry-columns.types';
import { assignColumns } from './masonry-columns.utils';

const childElements = (container: HTMLElement) =>
  Array.from(container.children).filter(
    (child): child is HTMLElement => child instanceof HTMLElement
  );

// Absolutely positions the container's children into columns and re-stacks them
// whenever one resizes or children are added/removed. Returns a stop function.
export const startMasonryLayout = (
  container: HTMLElement,
  { columnCount, gap }: MasonryLayoutOptions
): (() => void) => {
  const width = `calc((100% - ${(columnCount - 1) * gap}px) / ${columnCount})`;
  // Columns within one gap of each other look level.
  const tolerance = gap;
  let placement = new Map<HTMLElement, number>();
  // Before the user touches anything, sizes change only because content is
  // still loading, so re-balance freely; after that nothing changes column.
  let frozen = false;
  const freeze = () => {
    frozen = true;
  };

  const layout = () => {
    const items = childElements(container);
    // Width first: an item's height depends on it.
    items.forEach((item) => {
      item.style.width = width;
    });
    const heights = items.map((item) => item.offsetHeight);
    const assigned = assignColumns(
      items.map((item, index) => ({
        height: heights[index] + gap,
        column: frozen ? placement.get(item) : undefined,
      })),
      columnCount,
      tolerance
    );
    placement = new Map(items.map((item, index) => [item, assigned[index]]));

    const tops = Array<number>(columnCount).fill(0);
    items.forEach((item, index) => {
      const column = assigned[index];
      item.style.insetInlineStart = `calc(${column} * (100% + ${gap}px) / ${columnCount})`;
      item.style.top = `${tops[column]}px`;
      tops[column] += heights[index] + gap;
    });
    container.style.height = `${Math.max(0, ...tops.map((top) => top - gap))}px`;
  };

  const resizeObserver = new ResizeObserver(layout);
  const observeItems = () => {
    resizeObserver.disconnect();
    childElements(container).forEach((item) => resizeObserver.observe(item));
  };
  const mutationObserver = new MutationObserver(() => {
    observeItems();
    layout();
  });

  layout();
  observeItems();
  mutationObserver.observe(container, { childList: true });
  container.addEventListener('pointerdown', freeze, { capture: true });
  container.addEventListener('keydown', freeze, { capture: true });

  return () => {
    resizeObserver.disconnect();
    mutationObserver.disconnect();
    container.removeEventListener('pointerdown', freeze, { capture: true });
    container.removeEventListener('keydown', freeze, { capture: true });
  };
};
