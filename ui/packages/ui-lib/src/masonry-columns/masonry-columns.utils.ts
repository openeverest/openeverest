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

import { MasonryItem } from './masonry-columns.types';

// Placed items keep their column, so an item never moves when a sibling grows.
// New ones go to the shortest column; columns within `tolerance` of it count as
// equally short (like CSS Grid Lanes' flow-tolerance), and among those the one
// with fewer items wins, so nearly level columns fill left to right.
export const assignColumns = (
  items: MasonryItem[],
  count: number,
  tolerance: number
): number[] => {
  const heights = Array<number>(count).fill(0);
  const sizes = Array<number>(count).fill(0);
  const place = (column: number, height: number) => {
    heights[column] += height;
    sizes[column] += 1;
    return column;
  };
  const pickColumn = () => {
    const lowest = Math.min(...heights);
    const candidates = heights.flatMap((height, column) =>
      height - lowest <= tolerance ? [column] : []
    );
    return candidates.reduce((best, column) =>
      sizes[column] < sizes[best] ? column : best
    );
  };

  items.forEach(({ column, height }) => {
    if (column !== undefined) place(column, height);
  });

  return items.map(
    ({ column, height }) => column ?? place(pickColumn(), height)
  );
};
